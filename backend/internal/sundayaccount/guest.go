package sundayaccount

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/uptrace/bun"
)

// A contact address is not an identity. Guest creation deliberately performs
// no lookup by typed email or phone and never attaches another browser's data.
func accountProjection(alias string) string {
	if alias != "" {
		alias += "."
	}
	return alias + `id,` + alias + `full_name,COALESCE(` + alias + `login_email,` + alias + `email,` + alias + `contact_email,'') AS email,COALESCE(` + alias + `phone,'') AS phone,(` + alias + `email IS NULL AND ` + alias + `password_hash IS NULL) AS is_guest,` + alias + `telegram_username,(` + alias + `telegram_user_id IS NOT NULL AND NOT ` + alias + `telegram_blocked) AS telegram_linked`
}

func normalizePhone(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	var digits strings.Builder
	for i, c := range raw {
		switch {
		case c >= '0' && c <= '9':
			digits.WriteRune(c)
		case c == '+' && i == 0:
		case c == ' ' || c == '-' || c == '(' || c == ')':
		default:
			return "", errors.New("Enter a valid phone number")
		}
	}
	if digits.Len() < 7 || digits.Len() > 15 {
		return "", errors.New("Enter a valid phone number with 7 to 15 digits")
	}
	return "+" + digits.String(), nil
}

func (h *Handler) startGuest(w http.ResponseWriter, r *http.Request) {
	var input struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.FullName = strings.TrimSpace(input.FullName)
	if !validName(input.FullName) {
		fail(w, 400, "Enter your full name")
		return
	}
	email, err := normalizeEmail(input.Email)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	token, err := randomToken()
	if err != nil {
		fail(w, 503, "Could not start a browser account")
		return
	}
	ipHash := h.digest("guest-ip", h.requestIP(r))
	var account Account
	err = h.db.RunInTx(r.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(?,0))`, "sunday-guest:"+ipHash); err != nil {
			return err
		}
		var counts struct{ Recent, Hourly int }
		if err := tx.NewRaw(`SELECT count(*) FILTER(WHERE created_at>NOW()-INTERVAL '1 minute') AS recent,count(*) AS hourly FROM sunday_open_accounts WHERE guest_ip_hash=? AND created_at>NOW()-INTERVAL '1 hour'`, ipHash).Scan(ctx, &counts); err != nil {
			return err
		}
		if counts.Recent >= 3 || counts.Hourly >= 30 {
			return errRateLimited
		}
		if err := tx.NewRaw(`INSERT INTO sunday_open_accounts(email,contact_email,phone,full_name,guest_ip_hash) VALUES(NULL,?,NULLIF(?,''),?,?) RETURNING `+accountProjection(""), email, phone, input.FullName, ipHash).Scan(ctx, &account); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO sunday_open_sessions(account_id,token_hash,expires_at) VALUES(?,?,NOW()+INTERVAL '30 days')`, account.ID, h.digest("session", token))
		return err
	})
	if errors.Is(err, errRateLimited) {
		w.Header().Set("Retry-After", "60")
		fail(w, 429, "Please wait before starting another browser account")
		return
	}
	if err != nil {
		fail(w, 503, "Could not start a browser account")
		return
	}
	h.setCookie(w, token, 30*24*3600)
	h.writeSession(w, &principal{Account: account, token: token})
}
