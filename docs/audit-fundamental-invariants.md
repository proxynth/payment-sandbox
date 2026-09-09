# Audit des invariants fondamentaux

## Périmètre et environnement

- Dépôt : `proxynth/payment-sandbox`
- Branche analysée initialement : `main`
- Commit initial : `c12c9f9768532ca7ca2209568982e686ebbf4ac1`
- Branche de correction : `audit/fundamental-invariants`
- Go : 1.25.7
- Persistance exercée : SQLite migré jusqu'à la version 11

Les commandes exécutées après corrections sont `go test ./...`, `go test -race ./...` et `go vet ./...`. Elles passent toutes.

## Modèle réellement exécuté

Les commandes HTTP de paiement écrivent l'état courant dans `payments`, émettent un événement dans `event_log`, puis créent les jobs webhook dans `scheduler_jobs`. Ces trois effets sont exécutés dans la même transaction SQLite. Les workers acquièrent un job de manière atomique, exécutent le handler, puis enregistrent son état terminal ou sa prochaine tentative.

Les scénarios de replay utilisent un registre de providers configuré par seed, une horloge virtuelle et des repositories isolés en mémoire. Ils ne modifient donc pas l'état runtime. Le runtime persiste désormais ses contrôles : endpoints webhook, scénarios enregistrés et heure virtuelle.

## Invariants vérifiés et corrections

| Invariant | Preuve exécutée | État |
| --- | --- | --- |
| Une commande HTTP idempotente ne produit qu'un effet | tests HTTP d'idempotence et table SQLite durable | garanti pour les routes couvertes |
| Paiement, événement et jobs webhook sont atomiques | tests de crash et d'échec de publication | garanti pour les transitions de paiement |
| Un job échoué peut être repris | `TestAuditFailedJobCanRetry` | garanti |
| Le cycle des jobs est conservé pour inspection | snapshots append-only `scheduler_job_audit` et test SQLite | garanti |
| Chaque tentative webhook laisse une trace durable | `webhook_delivery_audit`, tests applicatifs et SQLite | garanti : résultat final ou marqueur `started` au résultat inconnu |
| Un crash après succès HTTP rend le doublon visible | `TestAuditWebhookCrashAfterSuccessBeforeCompletionIsTraceable` | garanti, livraison externe at-least-once |
| Deux workers n'obtiennent pas le même job | `TestAuditConcurrentAcquisitionHasOneWinner` et `-race` | garanti par acquisition SQL conditionnelle |
| Les leases expirés sont récupérés | `TestAuditExpiredJobsAreDiscovered` | garanti |
| L'event log distingue les états monétaires | `TestAuditEventLogDistinguishesAmounts` | garanti pour les nouveaux événements |
| Une Saga reprend après mutation paiement et échec de sauvegarde | `TestAuditSagaRecoversAfterPaymentCommit` | garanti pour les étapes reconnues comme déjà appliquées |
| Endpoint, scénario et temps virtuel survivent au redémarrage | `TestAuditRuntimeRestartPreservesControlState` | garanti |
| Un replay de scénario est isolé et reproductible | tests du runner, scénarios et comparaison | garanti dans le périmètre du runner |

## Écarts corrigés

1. Les mutations paiement directes pouvaient laisser un état courant sans événement si elles n'étaient pas invoquées par la couche HTTP. Elles ouvrent désormais une transaction via le repository SQLite.
2. Les jobs `failed` étaient listés mais ne pouvaient pas être acquis. L'acquisition accepte maintenant cet état de façon atomique.
3. L'event log ne contenait ni montant capturé ni montant remboursé. La migration 8 ajoute un payload immuable avec le snapshot métier.
4. Une nouvelle livraison de message Saga après un commit paiement pouvait rejouer une transition invalide. L'exécuteur reconnaît un état déjà atteint.
5. Les contrôles du runtime étaient purement mémoire. La migration 9 les rend durables.
6. Les tentatives webhook ne conservaient pas leur résultat protocolaire. La migration 11 enregistre, pour chaque paire `(job_id, tentative)`, le endpoint, la corrélation, le statut HTTP ou l'erreur de transport, sans persister les corps.

## Limites résiduelles

- Le replay reproduit les résultats métier qu'il expose, mais il ne reconstruit pas un runtime historique à partir de `event_log` et `scheduler_jobs`.
- L'audit webhook conserve le code HTTP et l'erreur de transport, mais pas les corps de requête ou de réponse : ils peuvent contenir des données sensibles et ne sont pas nécessaires au diagnostic initial. Une entrée `started` sans résultat final signale une tentative dont le résultat n'a pas pu être durablement enregistré (crash ou échec d'écriture après l’appel HTTP) ; elle ne permet pas d'affirmer si le destinataire a reçu le message.
- Une requête runtime sans `X-Correlation-ID` reçoit un identifiant dérivé de façon stable de sa méthode, son chemin, sa query et son corps. Le client peut toujours fournir sa propre valeur pour rattacher plusieurs requêtes à une même trace.
- SQLite apporte l'atomicité locale testée ici. Le projet ne revendique pas de disponibilité ou de coordination multi-processus au-delà de ses verrous SQLite.

## Verdict

Les propriétés qui étaient seulement déclarées au commit initial ne le sont plus toutes : les chemins transactionnels, l'idempotence HTTP, les retries, la concurrence de job, la persistance du contrôle runtime et les snapshots d'événements sont maintenant couverts par des tests exécutés.

Le déterminisme est réel pour les scénarios de replay et leurs résultats métier, ainsi que pour les métadonnées de corrélation des requêtes runtime équivalentes. L'audit couvre les transitions internes des jobs et le résultat de chaque livraison webhook. Une livraison externe demeure naturellement *at-least-once* : après un crash, le journal permet de diagnostiquer un doublon potentiel mais ne peut pas transformer HTTP en exactly-once.

L'audit de livraison est consultable avec le jeton d'administration via `GET /admin/webhook-jobs/{jobId}/deliveries`. La réponse expose l'identité de la tentative, le endpoint, les corrélations, le résultat, le statut HTTP et l'erreur éventuelle, sans exposer les corps de callback.
