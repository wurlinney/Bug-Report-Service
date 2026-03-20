package bootstrap

import (
	"context"
	"strings"

	"bug-report-service/internal/adapters/config"
	"bug-report-service/internal/adapters/observability"
	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/adapters/security"
)

func seedModerators(ctx context.Context, log observability.Logger, repo *postgres.ModeratorRepository, hasher security.PasswordHasher, seeds []config.ModeratorSeed) {
	if len(seeds) == 0 || repo == nil || hasher == nil {
		return
	}

	for _, s := range seeds {
		email := strings.ToLower(strings.TrimSpace(s.Email))
		if email == "" {
			continue
		}

		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = email
		}

		hash := strings.TrimSpace(s.PasswordHash)
		if hash == "" {
			pw := strings.TrimSpace(s.Password)
			if pw == "" {
				log.Error("moderator seed skipped: missing password/password_hash", "email", email)
				continue
			}
			h, err := hasher.HashPassword(pw)
			if err != nil {
				log.Error("moderator seed failed: hash password", "email", email, "error", err.Error())
				continue
			}
			hash = h
		}

		if err := repo.UpsertByEmail(ctx, name, email, hash); err != nil {
			log.Error("moderator seed failed: upsert", "email", email, "error", err.Error())
			continue
		}
		log.Info("moderator seeded", "email", email)
	}
}
