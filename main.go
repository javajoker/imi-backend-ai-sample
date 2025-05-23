// Project Structure Overview
/*
imi-backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── database.go
│   ├── models/
│   │   ├── user.go
│   │   ├── ip_asset.go
│   │   ├── license.go
│   │   ├── product.go
│   │   ├── transaction.go
│   │   ├── admin.go
│   │   └── common.go
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── ip_asset.go
│   │   ├── license.go
│   │   ├── product.go
│   │   ├── payment.go
│   │   ├── admin.go
│   │   └── verification.go
│   ├── services/
│   │   ├── auth_service.go
│   │   ├── user_service.go
│   │   ├── ip_service.go
│   │   ├── license_service.go
│   │   ├── product_service.go
│   │   ├── payment_service.go
│   │   ├── admin_service.go
│   │   ├── verification_service.go
│   │   ├── blockchain_service.go
│   │   └── notification_service.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── rate_limit.go
│   │   ├── i18n.go
│   │   └── logging.go
│   ├── database/
│   │   ├── connection.go
│   │   ├── migrations/
│   │   └── seeds/
│   ├── i18n/
│   │   ├── i18n.go
│   │   ├── locales/
│   │   │   ├── en.json
│   │   │   └── zh_TW.json
│   │   └── keys.go
│   ├── utils/
│   │   ├── jwt.go
│   │   ├── validator.go
│   │   ├── crypto.go
│   │   ├── pagination.go
│   │   └── response.go
│   └── router/
│       └── router.go
├── pkg/
│   ├── blockchain/
│   ├── storage/
│   └── events/
├── migrations/
├── docs/
├── scripts/
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
*/

package main

// This file shows the project structure and main entry point
// The actual implementation will be in separate files as shown in the structure above
