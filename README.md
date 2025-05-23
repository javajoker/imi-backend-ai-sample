# IP Marketplace Infrastructure Backend

The world's first open IP authorization and secondary creation marketplace platform that enables creators to monetize their intellectual property through transparent licensing while allowing secondary creators to build authentic products with verified authorization.

## 🌟 Features

### Core Functionality
- **IP Asset Management**: Register, verify, and manage intellectual property assets
- **License Management**: Apply for, approve, and manage licenses with automated workflows
- **Product Creation**: Create and sell products using licensed IP with authenticity verification
- **Payment Processing**: Secure payments with automatic revenue sharing
- **Blockchain Integration**: Immutable authorization chains for authenticity verification
- **Multi-language Support**: English and Traditional Chinese with extensible i18n system

### User Roles
- **IP Creators**: Artists, designers, and content creators who own intellectual property
- **Secondary Creators**: Manufacturers and sellers who create products using licensed IP
- **Buyers**: End consumers purchasing authentic licensed products
- **Administrators**: Platform operators managing the ecosystem

### Admin Features
- **Dashboard**: Real-time platform statistics and analytics
- **User Management**: Manage users, verification levels, and account status
- **Content Moderation**: Review and approve IP assets and handle reports
- **Transaction Management**: Monitor payments, process refunds, and manage disputes
- **Analytics**: Comprehensive business intelligence and reporting

## 🛠 Technology Stack

- **Language**: Go 1.21+
- **Framework**: Gin Web Framework
- **Database**: PostgreSQL with GORM
- **Cache**: Redis
- **Authentication**: JWT with refresh tokens
- **File Storage**: AWS S3 + CloudFront
- **Payments**: Stripe integration
- **Blockchain**: Ethereum/Polygon (configurable)
- **Email**: SMTP integration
- **Documentation**: Swagger/OpenAPI
- **Containerization**: Docker & Docker Compose

## 📋 Prerequisites

- Go 1.21 or later
- PostgreSQL 12+
- Redis 6+
- Docker & Docker Compose (optional but recommended)

## 🚀 Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/imi-backend.git
cd imi-backend
```

### 2. Setup Environment

```bash
# Copy environment template
cp .env.example .env

# Edit configuration (required)
nano .env
```

### 3. Run Setup Script

```bash
# Make scripts executable and run setup
chmod +x scripts/setup.sh
./scripts/setup.sh
```

### 4. Start with Docker (Recommended)

```bash
# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f backend
```

### 5. Or Run Locally

```bash
# Start PostgreSQL and Redis separately, then:
go run ./cmd/server

# Or build and run
make build
./bin/ip-marketplace
```

## 📊 API Documentation

The API follows RESTful principles with comprehensive endpoints for all platform features.

### Base URL
```
http://localhost:8080/v1
```

### Authentication
Most endpoints require JWT authentication via Authorization header:
```
Authorization: Bearer <your_jwt_token>
```

### Key Endpoints

#### Authentication
- `POST /auth/register` - User registration
- `POST /auth/login` - User login  
- `POST /auth/refresh` - Refresh JWT token
- `GET /auth/me` - Get current user profile

#### IP Assets
- `GET /ip-assets` - Browse IP assets with filters
- `POST /ip-assets` - Create new IP asset
- `GET /ip-assets/:id` - Get IP asset details
- `POST /ip-assets/:id/licenses` - Create license terms

#### Licenses
- `POST /licenses/apply` - Apply for license
- `GET /licenses/applications` - Get license applications
- `PUT /licenses/:id/approve` - Approve license (IP creator)
- `PUT /licenses/:id/reject` - Reject license (IP creator)

#### Products
- `GET /products` - Browse products with filters
- `POST /products` - Create new product (requires license)
- `POST /products/:id/purchase` - Purchase product
- `GET /products/:id/verify` - Verify product authenticity

#### Payments
- `POST /payments/intent` - Create payment intent
- `POST /payments/confirm` - Confirm payment
- `GET /payments/history` - Get payment history
- `GET /payments/balance` - Get user balance

#### Admin (Admin only)
- `GET /admin/dashboard/stats` - Platform statistics
- `GET /admin/users` - Manage users
- `GET /admin/ip-assets/pending` - Review pending IP assets
- `PUT /admin/ip-assets/:id/approve` - Approve IP asset
- `GET /admin/transactions` - Monitor transactions

#### Verification (Public)
- `GET /verify/:code` - Verify product by code
- `GET /verify/chain/:id` - Verify authorization chain

## 🔧 Configuration

### Environment Variables

Key configuration options in `.env`:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=ip_marketplace

# JWT Security
JWT_SECRET=your-super-secret-key

# AWS S3 (for file storage)
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_S3_BUCKET=your-bucket-name

# Stripe Payments
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password
```

## 🗄 Database Schema

The platform uses PostgreSQL with the following key entities:

- **Users**: Platform users with different roles
- **IP Assets**: Registered intellectual property
- **License Terms**: Licensing conditions for IP assets
- **License Applications**: Applications to use IP assets
- **Products**: Items created using licensed IP
- **Transactions**: Payment and revenue sharing records
- **Authorization Chains**: Blockchain-verified authenticity records

## 🔐 Security Features

- **JWT Authentication**: Secure token-based authentication
- **Rate Limiting**: API rate limiting to prevent abuse
- **Input Validation**: Comprehensive request validation
- **SQL Injection Prevention**: Parameterized queries
- **File Upload Security**: Type validation and virus scanning
- **Audit Logging**: Complete activity tracking
- **CORS Protection**: Configurable cross-origin policies

## 🌍 Internationalization

The platform supports multiple languages with easy extensibility:

- **English** (default)
- **Traditional Chinese**
- **Extensible**: Easy to add new languages

Language files are located in `internal/i18n/locales/`.

## 📱 API Client Examples

### JavaScript/TypeScript
```javascript
// Login
const response = await fetch('/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, password })
});

// Create IP Asset
const ipAsset = await fetch('/v1/ip-assets', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    title: 'My Artwork',
    description: 'Beautiful digital art',
    category: 'art'
  })
});
```

### cURL
```bash
# Login
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'

# Get IP Assets
curl -X GET "http://localhost:8080/v1/ip-assets?category=art&limit=10" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 🐳 Docker Deployment

### Development
```bash
docker-compose up -d
```

### Production
```bash
# Build production image
docker build -t imi-backend .

# Run with production settings
docker run -d \
  --name imi-backend \
  -p 8080:8080 \
  --env-file .env.production \
  imi-backend
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/services/... -v
```

## 📈 Monitoring & Analytics

The platform includes comprehensive monitoring:

- **Health Checks**: `/health` endpoint for service monitoring
- **Metrics**: Business and technical metrics collection
- **Audit Logs**: Complete user activity tracking
- **Analytics**: Platform usage and revenue analytics
- **Error Tracking**: Comprehensive error logging

## 🔄 Development Workflow

### Local Development
```bash
# Install development tools
make install-tools

# Run with live reload
make dev

# Format code
make fmt

# Run linter
make lint
```

### Database Migrations
```bash
# Run migrations
make migrate

# Or manually
./scripts/migrate.sh
```

## 📚 Additional Documentation

- [API Documentation](docs/api.md)
- [Database Schema](docs/database.md)
- [Deployment Guide](docs/deployment.md)
- [Contributing Guidelines](CONTRIBUTING.md)

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the Apache-2.0 License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

For support and questions:

- **Issues**: Create a GitHub issue
- **Email**: support@ipmarketplace.com
- **Documentation**: Check the `/docs` folder
- **Community**: Join our Discord server

## 🗺 Roadmap

### Phase 1 (Current)
- ✅ Core IP asset management
- ✅ License application system
- ✅ Product creation and sales
- ✅ Payment processing
- ✅ Admin dashboard

### Phase 2 (Q2 2024)
- 🔄 Mobile applications
- 🔄 Advanced analytics
- 🔄 API for third parties
- 🔄 Multi-language expansion

### Phase 3 (Q3 2024)
- 📋 AI-powered recommendations
- 📋 Global marketplace expansion
- 📋 Enterprise features
- 📋 Advanced blockchain integration

## 📊 Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   API Gateway   │    │   Database      │
│   (React)       │◄──►│   (Gin/Go)      │◄──►│   (PostgreSQL)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   Microservices │
                    │   - Auth        │
                    │   - IP Assets   │
                    │   - Licenses    │
                    │   - Products    │
                    │   - Payments    │
                    │   - Admin       │
                    └─────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Redis     │    │   AWS S3    │    │ Blockchain  │
│   (Cache)   │    │  (Storage)  │    │ (Ethereum)  │
└─────────────┘    └─────────────┘    └─────────────┘
```

---

**Built with ❤️ for the creator economy**
