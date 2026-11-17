//go:build ignore

// Command oldside runs inside a go-modules checkout of d6a572f, the last
// version that shipped stacks/saascore. run.sh copies it there; it is never
// built in this repository. It creates the legacy schema exactly as saascore
// did at startup and signs session tokens with the legacy signer.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	authjwt "github.com/brizenchi/go-modules/modules/auth/adapter/jwt"
	authdomain "github.com/brizenchi/go-modules/modules/auth/domain"
	"github.com/brizenchi/go-modules/stacks/saascore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: oldside schema|token [flags]")
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	dsn := fs.String("dsn", "", "Postgres DSN")
	secret := fs.String("secret", "", "legacy auth.user_jwt_secret")
	service := fs.String("service", "", "legacy server.name, used as the JWT issuer")
	userID := fs.String("user-id", "", "token subject")
	email := fs.String("email", "", "token email")
	role := fs.String("role", "user", "token role")
	if err := fs.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}

	switch os.Args[1] {
	case "schema":
		db, err := gorm.Open(postgres.Open(*dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("open db: %v", err)
		}
		// saascore.New runs every legacy AutoMigrate before wiring modules.
		cfg := saascore.Config{ServiceName: *service, Auth: saascore.AuthConfig{UserJWTSecret: *secret}}
		if _, err := saascore.New(db, cfg, saascore.HostHooks{}, saascore.PolicyHooks{}); err != nil {
			log.Fatalf("saascore.New: %v", err)
		}
	case "token":
		signer, err := authjwt.NewSigner(authjwt.Config{Secret: *secret, Issuer: *service})
		if err != nil {
			log.Fatal(err)
		}
		token, err := signer.Issue(authdomain.Identity{UserID: *userID, Email: *email, Role: authdomain.Role(*role)}, 0)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(token.Value)
	default:
		log.Fatalf("unknown mode %q", os.Args[1])
	}
}
