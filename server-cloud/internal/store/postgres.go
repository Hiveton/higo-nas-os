package store

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// Postgres is the production Store backend. It applies the schema idempotently on
// Connect, so there is no separate migration step for this single-file schema.
type Postgres struct {
	pool *pgxpool.Pool
}

// ConnectPostgres dials the DSN, verifies connectivity and applies the schema.
func ConnectPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := pool.Exec(ctx, schemaSQL); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

// Close releases the connection pool.
func (p *Postgres) Close() { p.pool.Close() }

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// --- Users & identities ------------------------------------------------------

func (p *Postgres) CreateUser(ctx context.Context, u User) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_users (id, display_name, avatar, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		u.ID, u.DisplayName, u.Avatar, u.Status, u.CreatedAt, u.UpdatedAt)
	return err
}

func (p *Postgres) GetUser(ctx context.Context, id string) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx,
		`SELECT id, display_name, avatar, status, created_at, updated_at FROM cloud_users WHERE id=$1`, id).
		Scan(&u.ID, &u.DisplayName, &u.Avatar, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	return u, mapErr(err)
}

func (p *Postgres) UpdateUser(ctx context.Context, u User) error {
	tag, err := p.pool.Exec(ctx,
		`UPDATE cloud_users SET display_name=$2, avatar=$3, status=$4, updated_at=$5 WHERE id=$1`,
		u.ID, u.DisplayName, u.Avatar, u.Status, u.UpdatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) CreateIdentity(ctx context.Context, i Identity) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_identities (id, user_id, type, principal, secret_hash, verified_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		i.ID, i.UserID, string(i.Type), i.Principal, i.SecretHash, i.VerifiedAt, i.CreatedAt)
	return err
}

func (p *Postgres) GetIdentity(ctx context.Context, t IdentityType, principal string) (Identity, error) {
	var i Identity
	var typ string
	err := p.pool.QueryRow(ctx,
		`SELECT id, user_id, type, principal, secret_hash, verified_at, created_at
		 FROM cloud_identities WHERE type=$1 AND principal=$2`, string(t), principal).
		Scan(&i.ID, &i.UserID, &typ, &i.Principal, &i.SecretHash, &i.VerifiedAt, &i.CreatedAt)
	i.Type = IdentityType(typ)
	return i, mapErr(err)
}

func (p *Postgres) ListIdentitiesByUser(ctx context.Context, userID string) ([]Identity, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, user_id, type, principal, secret_hash, verified_at, created_at
		 FROM cloud_identities WHERE user_id=$1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Identity
	for rows.Next() {
		var i Identity
		var typ string
		if err := rows.Scan(&i.ID, &i.UserID, &typ, &i.Principal, &i.SecretHash, &i.VerifiedAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		i.Type = IdentityType(typ)
		out = append(out, i)
	}
	return out, rows.Err()
}

func (p *Postgres) UpdateIdentity(ctx context.Context, i Identity) error {
	tag, err := p.pool.Exec(ctx,
		`UPDATE cloud_identities SET secret_hash=$2, verified_at=$3 WHERE id=$1`,
		i.ID, i.SecretHash, i.VerifiedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Refresh tokens ----------------------------------------------------------

func (p *Postgres) CreateRefreshToken(ctx context.Context, t RefreshToken) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_refresh_tokens (token_hash, id, user_id, expires_at, revoked_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		t.TokenHash, t.ID, t.UserID, t.ExpiresAt, t.RevokedAt, t.CreatedAt)
	return err
}

func (p *Postgres) GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error) {
	var t RefreshToken
	err := p.pool.QueryRow(ctx,
		`SELECT token_hash, id, user_id, expires_at, revoked_at, created_at
		 FROM cloud_refresh_tokens WHERE token_hash=$1`, tokenHash).
		Scan(&t.TokenHash, &t.ID, &t.UserID, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	return t, mapErr(err)
}

func (p *Postgres) RevokeRefreshToken(ctx context.Context, id string, at time.Time) error {
	_, err := p.pool.Exec(ctx, `UPDATE cloud_refresh_tokens SET revoked_at=$2 WHERE id=$1 AND revoked_at IS NULL`, id, at)
	return err
}

func (p *Postgres) RevokeUserRefreshTokens(ctx context.Context, userID string, at time.Time) error {
	_, err := p.pool.Exec(ctx, `UPDATE cloud_refresh_tokens SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL`, userID, at)
	return err
}

// --- Devices -----------------------------------------------------------------

func (p *Postgres) CreateDevice(ctx context.Context, d Device) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_devices (id, serial, secret_hash, model, version, agent_public_key, created_at, last_seen_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		d.ID, d.Serial, d.SecretHash, d.Model, d.Version, d.AgentPublicKey, d.CreatedAt, d.LastSeenAt)
	return err
}

func (p *Postgres) GetDevice(ctx context.Context, id string) (Device, error) {
	return p.scanDevice(ctx, `SELECT id, serial, secret_hash, model, version, agent_public_key, created_at, last_seen_at FROM cloud_devices WHERE id=$1`, id)
}

func (p *Postgres) GetDeviceBySerial(ctx context.Context, serial string) (Device, error) {
	return p.scanDevice(ctx, `SELECT id, serial, secret_hash, model, version, agent_public_key, created_at, last_seen_at FROM cloud_devices WHERE serial=$1`, serial)
}

func (p *Postgres) scanDevice(ctx context.Context, query, arg string) (Device, error) {
	var d Device
	err := p.pool.QueryRow(ctx, query, arg).
		Scan(&d.ID, &d.Serial, &d.SecretHash, &d.Model, &d.Version, &d.AgentPublicKey, &d.CreatedAt, &d.LastSeenAt)
	return d, mapErr(err)
}

func (p *Postgres) UpdateDevice(ctx context.Context, d Device) error {
	tag, err := p.pool.Exec(ctx,
		`UPDATE cloud_devices SET serial=$2, secret_hash=$3, model=$4, version=$5, agent_public_key=$6, last_seen_at=$7 WHERE id=$1`,
		d.ID, d.Serial, d.SecretHash, d.Model, d.Version, d.AgentPublicKey, d.LastSeenAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Bindings ----------------------------------------------------------------

func (p *Postgres) CreateBinding(ctx context.Context, b Binding) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_bindings (user_id, device_id, id, role, local_user_id, label, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		b.UserID, b.DeviceID, b.ID, b.Role, b.LocalUserID, b.Label, b.CreatedAt)
	return err
}

func (p *Postgres) GetBinding(ctx context.Context, userID, deviceID string) (Binding, error) {
	var b Binding
	err := p.pool.QueryRow(ctx,
		`SELECT id, user_id, device_id, role, local_user_id, label, created_at
		 FROM cloud_bindings WHERE user_id=$1 AND device_id=$2`, userID, deviceID).
		Scan(&b.ID, &b.UserID, &b.DeviceID, &b.Role, &b.LocalUserID, &b.Label, &b.CreatedAt)
	return b, mapErr(err)
}

func (p *Postgres) ListBindingsByUser(ctx context.Context, userID string) ([]Binding, error) {
	return p.queryBindings(ctx, `SELECT id, user_id, device_id, role, local_user_id, label, created_at FROM cloud_bindings WHERE user_id=$1 ORDER BY created_at`, userID)
}

func (p *Postgres) ListBindingsByDevice(ctx context.Context, deviceID string) ([]Binding, error) {
	return p.queryBindings(ctx, `SELECT id, user_id, device_id, role, local_user_id, label, created_at FROM cloud_bindings WHERE device_id=$1`, deviceID)
}

func (p *Postgres) queryBindings(ctx context.Context, query, arg string) ([]Binding, error) {
	rows, err := p.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Binding
	for rows.Next() {
		var b Binding
		if err := rows.Scan(&b.ID, &b.UserID, &b.DeviceID, &b.Role, &b.LocalUserID, &b.Label, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (p *Postgres) DeleteBinding(ctx context.Context, userID, deviceID string) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM cloud_bindings WHERE user_id=$1 AND device_id=$2`, userID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Pairing challenges ------------------------------------------------------

func (p *Postgres) PutPairingChallenge(ctx context.Context, c PairingChallenge) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_pairing_challenges (kind, secret, device_id, expires_at, claimed)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (kind, secret) DO UPDATE SET device_id=EXCLUDED.device_id, expires_at=EXCLUDED.expires_at, claimed=EXCLUDED.claimed`,
		string(c.Kind), c.Secret, c.DeviceID, c.ExpiresAt, c.Claimed)
	return err
}

func (p *Postgres) GetPairingChallenge(ctx context.Context, kind PairingKind, secret string) (PairingChallenge, error) {
	var c PairingChallenge
	var k string
	err := p.pool.QueryRow(ctx,
		`SELECT kind, secret, device_id, expires_at, claimed FROM cloud_pairing_challenges WHERE kind=$1 AND secret=$2`,
		string(kind), secret).Scan(&k, &c.Secret, &c.DeviceID, &c.ExpiresAt, &c.Claimed)
	c.Kind = PairingKind(k)
	return c, mapErr(err)
}

func (p *Postgres) MarkPairingClaimed(ctx context.Context, kind PairingKind, secret string) error {
	tag, err := p.pool.Exec(ctx, `UPDATE cloud_pairing_challenges SET claimed=TRUE WHERE kind=$1 AND secret=$2`, string(kind), secret)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Push tokens -------------------------------------------------------------

func (p *Postgres) UpsertPushToken(ctx context.Context, t PushToken) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_push_tokens (user_id, token, id, platform, created_at)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (user_id, token) DO UPDATE SET platform=EXCLUDED.platform`,
		t.UserID, t.Token, t.ID, t.Platform, t.CreatedAt)
	return err
}

func (p *Postgres) ListPushTokensByUser(ctx context.Context, userID string) ([]PushToken, error) {
	rows, err := p.pool.Query(ctx, `SELECT user_id, token, id, platform, created_at FROM cloud_push_tokens WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PushToken
	for rows.Next() {
		var t PushToken
		if err := rows.Scan(&t.UserID, &t.Token, &t.ID, &t.Platform, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// --- Audit -------------------------------------------------------------------

func (p *Postgres) AppendAudit(ctx context.Context, e AuditEntry) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO cloud_audit (id, user_id, device_id, action, detail, ip, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.UserID, e.DeviceID, e.Action, e.Detail, e.IP, e.CreatedAt)
	return err
}

func (p *Postgres) ListAuditByUser(ctx context.Context, userID string, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := p.pool.Query(ctx,
		`SELECT id, user_id, device_id, action, detail, ip, created_at
		 FROM cloud_audit WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.DeviceID, &e.Action, &e.Detail, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

var _ Store = (*Postgres)(nil)
