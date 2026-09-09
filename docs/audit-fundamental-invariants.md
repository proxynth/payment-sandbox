# Audit des invariants fondamentaux

## Périmètre et environnement

- Dépôt : `proxynth/payment-sandbox`
- Branche analysée initialement : `main`
- Commit initial : `c12c9f9768532ca7ca2209568982e686ebbf4ac1`
- Branche de correction : `audit/fundamental-invariants`
- Go : 1.25.7
- Persistance exercée : SQLite migré jusqu'à la version 13
- Dernier commit d'audit publié : `50942ad8dd9df67c87f715ecbd7fcf3b15d3008b`

Les commandes exécutées après corrections sont `go test ./...`, `go test -race ./...` et `go vet ./...`. Elles passent toutes.

## Modèle réellement exécuté

Les commandes HTTP de paiement écrivent l'état courant dans `payments`, émettent un événement dans `event_log`, puis créent les jobs webhook dans `scheduler_jobs`. Ces trois effets sont exécutés dans la même transaction SQLite. Les workers acquièrent un job de manière atomique, exécutent le handler, puis enregistrent son état terminal ou sa prochaine tentative.

Les scénarios de replay utilisent un registre de providers configuré par seed, une horloge virtuelle et des repositories isolés en mémoire. Ils ne modifient donc pas l'état runtime. Le runtime persiste désormais ses contrôles : endpoints webhook, scénarios enregistrés et heure virtuelle.

## Invariants vérifiés et corrections

| Invariant | Preuve exécutée | État |
| --- | --- | --- |
| Une commande HTTP idempotente ne produit qu'un effet | tests HTTP d'idempotence et table SQLite durable | vérifié dans les tests cités pour les routes couvertes |
| Paiement, événement et jobs webhook sont atomiques | tests de crash et d'échec de publication | vérifié dans les tests cités pour les transitions de paiement |
| Un job échoué peut être repris | `TestAuditFailedJobCanRetry` | vérifié dans les tests cités |
| Le cycle des jobs est conservé et restaurable | snapshots append-only `scheduler_job_audit` et test SQLite | vérifié dans les tests cités pour les snapshots créés après migration 12 |
| Un job est relié à sa cause métier | métadonnées `aggregate_id` / `causation_id` persistées | vérifié dans les tests cités pour les nouveaux jobs créés après migration 13 |
| Chaque tentative webhook laisse une trace durable | `webhook_delivery_audit`, tests applicatifs et SQLite | vérifié dans les tests cités : résultat final ou marqueur `started` au résultat inconnu |
| Un crash après succès HTTP rend le doublon visible | `TestAuditWebhookCrashAfterSuccessBeforeCompletionIsTraceable` | vérifié dans les tests cités, livraison externe at-least-once |
| Deux workers n'obtiennent pas le même job | `TestAuditConcurrentAcquisitionHasOneWinner` et `-race` | vérifié dans les tests cités par acquisition SQL conditionnelle |
| Les leases expirés sont récupérés | `TestAuditExpiredJobsAreDiscovered` | vérifié dans les tests cités |
| L'event log distingue les états monétaires | `TestAuditEventLogDistinguishesAmounts` | vérifié dans les tests cités pour les nouveaux événements |
| L'event log reconstruit l'état courant | `TestAuditEventLogReconstructsCurrentPaymentState` | vérifié dans les tests cités pour les nouveaux historiques complets |
| Une Saga reprend après mutation paiement et échec de sauvegarde | `TestAuditSagaRecoversAfterPaymentCommit` | vérifié dans les tests cités pour les étapes reconnues comme déjà appliquées |
| Endpoint, scénario et temps virtuel survivent au redémarrage | `TestAuditRuntimeRestartPreservesControlState` | vérifié dans les tests cités |
| Un replay de scénario est isolé et reproductible | tests du runner, scénarios et comparaison | vérifié dans les tests cités dans le périmètre du runner |
| L'historique runtime global est consultable sans effet de bord | `RuntimeHistory`, `ListAuditByAggregate`, test de reconstruction | vérifié dans les tests cités pour les événements/jobs créés avec causalité persistée |

## Écarts corrigés

1. Les mutations paiement directes pouvaient laisser un état courant sans événement si elles n'étaient pas invoquées par la couche HTTP. Elles ouvrent désormais une transaction via le repository SQLite.
2. Les jobs `failed` étaient listés mais ne pouvaient pas être acquis. L'acquisition accepte maintenant cet état de façon atomique.
3. L'event log ne contenait ni montant capturé ni montant remboursé. La migration 8 ajoute un payload immuable avec le snapshot métier.
4. Une nouvelle livraison de message Saga après un commit paiement pouvait rejouer une transition invalide. L'exécuteur reconnaît un état déjà atteint.
5. Les contrôles du runtime étaient purement mémoire. La migration 9 les rend durables.
6. Les tentatives webhook ne conservaient pas leur résultat protocolaire. La migration 11 enregistre, pour chaque paire `(job_id, tentative)`, le endpoint, la corrélation, le statut HTTP ou l'erreur de transport, sans persister les corps.
7. Il manquait une vue de corrélation entre événement métier, cycle du job et livraison webhook. La migration 13 persiste `aggregate_id`/`causation_id` et l'administration expose `GET /admin/runtime-history/payments/{paymentId}` en lecture seule.

## Findings adversariaux

| Gravité | Finding | Preuve / impact | Correction ou limite |
| --- | --- | --- | --- |
| Medium | Les jobs antérieurs à la migration 13 ne sont pas causalement rattachables de façon fiable | Le schéma historique ne contenait pas ces colonnes ; une reconstruction globale peut donc omettre ces jobs | Limite explicitement exposée ; migration des nouvelles écritures, pas de fausse rétro-inférence |
| Medium | La livraison HTTP reste at-least-once | Crash possible après succès distant et avant l'accusé durable ; doublon externe possible | Marqueur `started`, résultat terminal et idempotence côté destinataire recommandée |
| Low | L'endpoint runtime history ne restitue pas les payloads de jobs | Diagnostic fonctionnel limité si le payload est nécessaire pour comprendre un cas précis | Choix de minimisation des données ; les événements et métadonnées causales restent consultables |

Les trois points ci-dessus sont des limites ou risques démontrés, pas des défaillances silencieuses découvertes après la correction. Aucun double paiement n'a été reproduit sur les chemins testés.

## Limites résiduelles

- Le replay de scénarios ne reconstitue pas encore automatiquement un runtime historique global. En revanche, l'état métier paiement est reconstructible depuis un historique complet d'`event_log` créé après la migration 8, et chaque job est restaurable depuis son historique `scheduler_job_audit` enrichi après la migration 12. Les nouveaux jobs portent en plus une cause métier explicite depuis la migration 13 ; les jobs antérieurs restent sans lien causal rétroactif fiable.
- L'audit webhook conserve le code HTTP et l'erreur de transport, mais pas les corps de requête ou de réponse : ils peuvent contenir des données sensibles et ne sont pas nécessaires au diagnostic initial. Une entrée `started` sans résultat final signale une tentative dont le résultat n'a pas pu être durablement enregistré (crash ou échec d'écriture après l’appel HTTP) ; elle ne permet pas d'affirmer si le destinataire a reçu le message.
- Une requête runtime sans `X-Correlation-ID` reçoit un identifiant dérivé de façon stable de sa méthode, son chemin, sa query et son corps. Le client peut toujours fournir sa propre valeur pour rattacher plusieurs requêtes à une même trace.
- SQLite apporte l'atomicité locale testée ici. Le projet ne revendique pas de disponibilité ou de coordination multi-processus au-delà de ses verrous SQLite.

## Verdict

Les propriétés qui étaient seulement déclarées au commit initial ne le sont plus toutes : les chemins transactionnels, l'idempotence HTTP, les retries, la concurrence de job, la persistance du contrôle runtime et les snapshots d'événements sont maintenant couverts par des tests exécutés.

Le déterminisme est réel pour les scénarios de replay et leurs résultats métier, ainsi que pour les métadonnées de corrélation des requêtes runtime équivalentes. L'audit couvre les transitions internes des jobs et le résultat de chaque livraison webhook. Une livraison externe demeure naturellement *at-least-once* : après un crash, le journal permet de diagnostiquer un doublon potentiel mais ne peut pas transformer HTTP en exactly-once.

L'audit de livraison est consultable avec le jeton d'administration via `GET /admin/webhook-jobs/{jobId}/deliveries`. La réponse expose l'identité de la tentative, le endpoint, les corrélations, le résultat, le statut HTTP et l'erreur éventuelle, sans exposer les corps de callback.

La vue consolidée est consultable avec le même jeton via `GET /admin/runtime-history/payments/{paymentId}`. Elle reconstruit l'état du paiement depuis l'event log et joint les snapshots de jobs et les tentatives webhook par causalité persistée ; elle n'exécute aucun handler et n'effectue aucun appel réseau.


## Contre-vérification du 10 septembre 2026 (modifications locales)

Le verdict précédent était trop affirmatif : des tests réussis ne démontrent pas
une garantie universelle. Le test applicatif de RuntimeHistory utilise des doubles
et vérifie essentiellement les nombres d'événements, jobs et livraisons ; il ne
prouve ni l'isolation des agrégats, ni l'absence d'écritures SQLite, ni la cohérence
d'une lecture concurrente.

### High — Blocage de la lecture des jobs dans l'historique global

Bug reproduit sur SQLite migré en version 13, avec la configuration réelle
`SetMaxOpenConns(1)`. `ListAuditByAggregate` conserve son curseur ouvert puis
appelle `ListAudit`, qui attend la connexion détenue par ce curseur. Avec un job
rattaché, le test `TestAggregateHistoryWithSingleConnection` échoue avant correction
avec `context deadline exceeded` après une seconde et réussit après correction.
Impact : consultation de l'historique indisponible et monopolisation de l'unique
connexion ; probabilité systématique pour ce scénario. Invariant affecté :
accessibilité de l'audit. Correction minimale : collecter les identifiants et
fermer le curseur avant de charger les snapshots, et respecter la transaction
éventuellement présente dans le contexte. Aucune migration nécessaire.

### Risques encore ouverts

- La vue globale fait plusieurs lectures sans transaction de lecture commune :
  une réponse peut mélanger des états observés à des instants différents si un
  worker ou une commande écrit entre les requêtes. Risque issu du code, non
  reproduit par un test concurrent à ce stade.
- Les tests HTTP dédiés à runtime-history, dont la non-exposition des payloads,
  restent à ajouter. L'absence de champ payload dans le DTO est un constat de code.
- L'ordre des snapshots est calculé à partir des tentatives, dates et statuts ;
  ce n'est pas une séquence persistée prouvant l'ordre réel de toutes les mutations.

Ces points empêchent de conclure que l'ensemble des propriétés fondamentales
est garanti. Les résultats positifs restent limités aux scénarios exécutés.

Validation de cette contre-vérification : `go test ./...`, `go test -race ./...`
et `go vet ./...` terminent avec succès. `make check` passe tidy, build et tests,
puis échoue sur `golangci-lint` absent ; `make fmt` échoue également à cette étape
après gofmt. Le contrôle obligatoire de lint reste donc non validé. Le correctif
et ce complément de rapport sont commités sur la branche d'audit ; le commit a
été publié après validation des tests disponibles.
