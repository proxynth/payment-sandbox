package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	paymentdomain "proxynth/payment-sandbox/internal/payment/domain"
	providerdomain "proxynth/payment-sandbox/internal/provider/domain"
	"proxynth/payment-sandbox/internal/replay/domain"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, scenario *domain.Scenario) error {
	if scenario == nil {
		return domain.ErrInvalidScenarioID
	}
	if err := scenario.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(toRecord(*scenario))
	if err != nil {
		return fmt.Errorf("encode scenario %q: %w", scenario.ID, err)
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO replay_scenarios(id,payload) VALUES (?,?) ON CONFLICT(id) DO NOTHING`, scenario.ID, string(payload))
	if err != nil {
		return fmt.Errorf("save scenario %q: %w", scenario.ID, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return domain.ErrScenarioAlreadyExists
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id domain.ScenarioID) (*domain.Scenario, error) {
	var payload []byte
	err := r.db.QueryRowContext(ctx, `SELECT payload FROM replay_scenarios WHERE id = ?`, id).Scan(&payload)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find scenario %q: %w", id, err)
	}
	var record scenarioRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return nil, fmt.Errorf("decode scenario %q: %w", id, err)
	}
	return record.toDomain()
}

type scenarioRecord struct {
	ID                 string          `json:"id"`
	InitialPayments    []paymentRecord `json:"initial_payments"`
	Commands           []commandRecord `json:"commands"`
	ProviderID         string          `json:"provider_id"`
	ProviderProfile    string          `json:"provider_profile"`
	InitialVirtualTime time.Time       `json:"initial_virtual_time"`
	Seed               uint64          `json:"seed"`
}
type paymentRecord struct {
	ID               string `json:"id"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	AuthorizedAmount int64  `json:"authorized_amount"`
	CapturedAmount   int64  `json:"captured_amount"`
	RefundedAmount   int64  `json:"refunded_amount"`
	Version          uint64 `json:"version"`
}
type commandRecord struct {
	Type          string `json:"type"`
	PaymentID     string `json:"payment_id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	DurationNanos int64  `json:"duration_nanos"`
	OperationID   string `json:"operation_id"`
}

func toRecord(s domain.Scenario) scenarioRecord {
	r := scenarioRecord{ID: string(s.ID), ProviderID: string(s.Provider.ID), ProviderProfile: s.Provider.Profile, InitialVirtualTime: s.InitialVirtualTime, Seed: s.DeterministicConfiguration.Seed}
	for _, p := range s.InitialPayments {
		r.InitialPayments = append(r.InitialPayments, paymentRecord{ID: string(p.ID), Amount: p.Amount.Amount(), Currency: string(p.Amount.Currency()), Status: string(p.Status), AuthorizedAmount: p.AuthorizedAmount, CapturedAmount: p.CapturedAmount, RefundedAmount: p.RefundedAmount, Version: p.Version})
	}
	for _, c := range s.Commands {
		r.Commands = append(r.Commands, commandRecord{Type: string(c.Type), PaymentID: string(c.PaymentID), Amount: c.Amount.Amount(), Currency: string(c.Amount.Currency()), DurationNanos: int64(c.Duration), OperationID: c.OperationID})
	}
	return r
}

func (r scenarioRecord) toDomain() (*domain.Scenario, error) {
	payments := make([]paymentdomain.PaymentState, 0, len(r.InitialPayments))
	for _, p := range r.InitialPayments {
		amount, err := paymentdomain.NewMoney(p.Amount, paymentdomain.Currency(p.Currency))
		if err != nil {
			return nil, err
		}
		payments = append(payments, paymentdomain.PaymentState{ID: paymentdomain.ID(p.ID), Amount: amount, Status: paymentdomain.Status(p.Status), AuthorizedAmount: p.AuthorizedAmount, CapturedAmount: p.CapturedAmount, RefundedAmount: p.RefundedAmount, Version: p.Version})
	}
	commands := make([]domain.Command, 0, len(r.Commands))
	for _, c := range r.Commands {
		amount := paymentdomain.Money{}
		if c.Amount != 0 || c.Currency != "" {
			var err error
			amount, err = paymentdomain.NewMoney(c.Amount, paymentdomain.Currency(c.Currency))
			if err != nil {
				return nil, err
			}
		}
		commands = append(commands, domain.Command{Type: domain.CommandType(c.Type), PaymentID: paymentdomain.ID(c.PaymentID), Amount: amount, Duration: time.Duration(c.DurationNanos), OperationID: c.OperationID})
	}
	return domain.New(domain.ScenarioID(r.ID), payments, commands, domain.ProviderConfiguration{ID: providerdomain.ProviderID(r.ProviderID), Profile: r.ProviderProfile}, r.InitialVirtualTime, domain.DeterministicConfiguration{Seed: r.Seed})
}
