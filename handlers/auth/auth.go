package handlers

import (
	"database/sql"
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Disposable email domains to block
var blockedDomains = map[string]bool{
	"mailinator.com":    true,
	"guerrillamail.com": true,
	"tempmail.com":      true,
	"throwaway.email":   true,
	// add more or pull from an open list
}

type SignupRequest struct {
	Email    string `json:"email"`
	Honeypot string `json:"website"` // bots fill this, humans don't see it
}

func Signup(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SignupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 1. Honeypot check — silently accept so bots think they succeeded
		if req.Honeypot != "" {
			c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
			return
		}

		// 2. Validate email format
		addr, err := mail.ParseAddress(req.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
			return
		}
		email := strings.ToLower(addr.Address)

		// 3. Block disposable domains
		domain := strings.Split(email, "@")[1]
		if blockedDomains[domain] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email domain not allowed"})
			return
		}

		// 4. Normalize email (strip Gmail dots and + aliases)
		email = normalizeEmail(email, domain)

		// 5. Insert in a transaction
		userID := uuid.New()

		tx, err := db.BeginTx(c, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not start transaction"})
			return
		}
		defer tx.Rollback()

		_, err = tx.ExecContext(c,
			`INSERT INTO users (id, email, created_at) VALUES ($1, $2, NOW())`,
			userID, email,
		)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				// 👇 this line — swap this depending on which behaviour you want

				// Option A — tells the user the email is taken (current)
				//c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})

				// Option B — silent accept, doesn't leak whether email exists
				c.JSON(http.StatusOK, gin.H{"message": "You're already on the waitlist!"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
			return
		}

		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not commit transaction"})
			return
		}

		// TODO: send confirmation/verification email here

		c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
	}
}

func normalizeEmail(email, domain string) string {
	parts := strings.Split(email, "@")
	local := parts[0]

	// Strip + alias (works for any provider)
	if idx := strings.Index(local, "+"); idx != -1 {
		local = local[:idx]
	}
	// Strip dots (Gmail ignores them)
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}

	return local + "@" + domain
}

func Counter(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var count int

		err := db.QueryRowContext(c, `SELECT COUNT(*) FROM users`).Scan(&count)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch count"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"count": count})
	}
}
