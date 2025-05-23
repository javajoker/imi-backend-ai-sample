# API Documentation

Complete API reference for the IP Marketplace Backend. All endpoints follow RESTful principles and return JSON responses.

## 📋 Table of Contents

- [Base Information](#base-information)
- [Authentication](#authentication)
- [Error Handling](#error-handling)
- [Pagination](#pagination)
- [Rate Limiting](#rate-limiting)
- [API Endpoints](#api-endpoints)
  - [Authentication](#authentication-endpoints)
  - [Users](#user-endpoints)
  - [IP Assets](#ip-asset-endpoints)
  - [Licenses](#license-endpoints)
  - [Products](#product-endpoints)
  - [Payments](#payment-endpoints)
  - [Verification](#verification-endpoints)
  - [Admin](#admin-endpoints)
- [Webhook Events](#webhook-events)
- [SDKs and Examples](#sdks-and-examples)

## Base Information

### Base URL
```
Production: https://api.ipmarketplace.com/v1
Staging: https://staging-api.ipmarketplace.com/v1
Development: http://localhost:8080/v1
```

### Content Type
All requests should include:
```
Content-Type: application/json
Accept: application/json
```

### Internationalization
Include language preference in headers:
```
Accept-Language: en          # English (default)
Accept-Language: zh-TW       # Traditional Chinese
```

### API Versioning
Current API version is `v1`. Future versions will be available at `/v2`, `/v3`, etc.

## Authentication

Most endpoints require authentication using JWT tokens. Include the token in the Authorization header:

```
Authorization: Bearer <your_jwt_token>
```

### Token Lifecycle
- **Access Token**: Expires in 24 hours
- **Refresh Token**: Expires in 7 days
- Use refresh endpoint to get new tokens without re-login

### User Roles
- **Creator**: Can create IP assets and license terms
- **Secondary Creator**: Can apply for licenses and create products
- **Buyer**: Can purchase products
- **Admin**: Can manage platform operations

## Error Handling

### Standard Error Response
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": {
      "field": "email",
      "reason": "Invalid email format"
    }
  }
}
```

### Error Codes
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid request data |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |
| `PAYMENT_FAILED` | 400 | Payment processing failed |
| `LICENSE_EXPIRED` | 400 | License has expired |

## Pagination

List endpoints support pagination with query parameters:

### Parameters
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20, max: 100)
- `sort`: Sort field (default: created_at)
- `order`: Sort order (asc/desc, default: desc)

### Response Headers
```
X-Total-Count: 150
X-Page: 1
X-Per-Page: 20
X-Total-Pages: 8
```

### Example Response
```json
{
  "success": true,
  "data": [...],
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 150,
      "total_pages": 8
    }
  }
}
```

## Rate Limiting

### Limits
- **General API**: 1000 requests/hour per user
- **Authentication**: 5 requests/minute per IP
- **File Upload**: 10 requests/minute per user

### Headers
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1642678800
```

## API Endpoints

## Authentication Endpoints

### Register User
Creates a new user account.

```
POST /auth/register
```

**Request Body:**
```json
{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "SecurePass123!",
  "user_type": "creator",
  "profile_data": {
    "first_name": "John",
    "last_name": "Doe",
    "bio": "Digital artist and designer"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Registration successful. Please check your email for verification.",
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "username": "johndoe",
      "email": "john@example.com",
      "user_type": "creator",
      "verification_level": "unverified",
      "status": "active",
      "created_at": "2024-01-15T10:30:00Z"
    },
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 86400
  }
}
```

### Login
Authenticates user and returns JWT tokens.

```
POST /auth/login
```

**Request Body:**
```json
{
  "email": "john@example.com",
  "password": "SecurePass123!"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Login successful",
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "username": "johndoe",
      "email": "john@example.com",
      "user_type": "creator"
    },
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 86400
  }
}
```

### Refresh Token
Gets new access token using refresh token.

```
POST /auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": { ... },
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "expires_in": 86400
  }
}
```

### Get Current User Profile
Returns current authenticated user's profile.

```
GET /auth/me
```
*Requires Authentication*

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "username": "johndoe",
      "email": "john@example.com",
      "user_type": "creator",
      "verification_level": "verified",
      "status": "active",
      "profile_data": {
        "first_name": "John",
        "last_name": "Doe",
        "bio": "Digital artist and designer",
        "avatar_url": "https://cdn.example.com/avatars/john.jpg"
      },
      "created_at": "2024-01-15T10:30:00Z",
      "last_login_at": "2024-01-20T15:45:00Z"
    }
  }
}
```

### Forgot Password
Sends password reset email.

```
POST /auth/forgot-password
```

**Request Body:**
```json
{
  "email": "john@example.com"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Password reset email sent"
  }
}
```

### Reset Password
Resets password using reset token.

```
POST /auth/reset-password
```

**Request Body:**
```json
{
  "token": "reset-token-from-email",
  "new_password": "NewSecurePass123!"
}
```

### Verify Email
Verifies email address using verification token.

```
GET /auth/verify-email/:token
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Email verified successfully"
  }
}
```

## User Endpoints

### Update Profile
Updates user profile information.

```
PUT /users/profile
```
*Requires Authentication*

**Request Body:**
```json
{
  "username": "johnsmith",
  "profile_data": {
    "first_name": "John",
    "last_name": "Smith",
    "bio": "Professional digital artist with 10+ years experience",
    "website": "https://johnsmith.art",
    "social_links": {
      "instagram": "@johnsmith_art",
      "twitter": "@johnsmith"
    }
  }
}
```

### Get User Profile
Gets public profile of any user.

```
GET /users/:id/public
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "username": "johndoe",
      "user_type": "creator",
      "verification_level": "verified",
      "profile_data": {
        "first_name": "John",
        "last_name": "Doe",
        "bio": "Digital artist and designer",
        "avatar_url": "https://cdn.example.com/avatars/john.jpg"
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Upload Avatar
Uploads user avatar image.

```
POST /users/upload-avatar
```
*Requires Authentication*
*Content-Type: multipart/form-data*

**Request:**
```
Form Data:
avatar: [image file]
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Avatar uploaded successfully",
    "avatar_url": "https://cdn.example.com/avatars/user-123.jpg"
  }
}
```

## IP Asset Endpoints

### Get IP Assets
Retrieves list of IP assets with filtering and pagination.

```
GET /ip-assets
```

**Query Parameters:**
- `category`: Filter by category (art, gaming, music, etc.)
- `creator_id`: Filter by creator
- `verification_status`: Filter by verification status
- `tags`: Comma-separated list of tags
- `search`: Search in title and description
- Standard pagination parameters

**Example Request:**
```
GET /ip-assets?category=art&verification_status=approved&page=1&limit=10
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "Digital Art Collection #1",
      "description": "Beautiful abstract digital artwork collection",
      "category": "art",
      "content_type": "image/png",
      "file_urls": [
        "https://cdn.example.com/ip-assets/artwork1.png",
        "https://cdn.example.com/ip-assets/artwork2.png"
      ],
      "tags": ["abstract", "digital", "art"],
      "verification_status": "approved",
      "status": "active",
      "view_count": 1250,
      "like_count": 89,
      "creator": {
        "id": "creator-id",
        "username": "artistjohn",
        "verification_level": "verified"
      },
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-16T14:20:00Z"
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 150,
      "total_pages": 15
    }
  }
}
```

### Create IP Asset
Creates a new IP asset.

```
POST /ip-assets
```
*Requires Authentication (Creator role)*

**Request Body:**
```json
{
  "title": "Digital Art Collection #1",
  "description": "Beautiful abstract digital artwork collection perfect for commercial use",
  "category": "art",
  "content_type": "image/png",
  "tags": ["abstract", "digital", "art", "commercial"],
  "metadata": {
    "resolution": "4096x4096",
    "file_format": "PNG",
    "color_profile": "sRGB",
    "usage_rights": "Commercial use allowed"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "IP asset created successfully",
    "ip_asset": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "Digital Art Collection #1",
      "description": "Beautiful abstract digital artwork collection perfect for commercial use",
      "category": "art",
      "verification_status": "pending",
      "status": "active",
      "creator_id": "creator-id",
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Get IP Asset Details
Retrieves detailed information about a specific IP asset.

```
GET /ip-assets/:id
```

**Response:**
```json
{
  "success": true,
  "data": {
    "ip_asset": {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "title": "Digital Art Collection #1",
      "description": "Beautiful abstract digital artwork collection",
      "category": "art",
      "content_type": "image/png",
      "file_urls": ["..."],
      "tags": ["abstract", "digital", "art"],
      "metadata": { ... },
      "verification_status": "approved",
      "status": "active",
      "view_count": 1250,
      "like_count": 89,
      "creator": { ... },
      "license_terms": [
        {
          "id": "license-terms-id",
          "license_type": "standard",
          "revenue_share_percentage": 15.0,
          "base_fee": 0,
          "territory": "global",
          "duration": "perpetual",
          "auto_approve": false
        }
      ],
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Update IP Asset
Updates an existing IP asset.

```
PUT /ip-assets/:id
```
*Requires Authentication (Creator - own assets only)*

**Request Body:**
```json
{
  "title": "Updated Digital Art Collection #1",
  "description": "Updated description with more details",
  "tags": ["abstract", "digital", "art", "premium"]
}
```

### Delete IP Asset
Deletes an IP asset (soft delete).

```
DELETE /ip-assets/:id
```
*Requires Authentication (Creator - own assets only)*

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "IP asset deleted successfully"
  }
}
```

### Create License Terms
Creates licensing terms for an IP asset.

```
POST /ip-assets/:id/licenses
```
*Requires Authentication (Creator - own assets only)*

**Request Body:**
```json
{
  "license_type": "standard",
  "revenue_share_percentage": 20.0,
  "base_fee": 50.0,
  "territory": "global",
  "duration": "perpetual",
  "requirements": "Must credit original creator",
  "restrictions": "Cannot be used for adult content",
  "auto_approve": false,
  "max_licenses": 100
}
```

### Upload IP Asset Files
Uploads files for an IP asset.

```
POST /ip-assets/upload
```
*Requires Authentication (Creator role)*
*Content-Type: multipart/form-data*

**Request:**
```
Form Data:
files: [multiple files]
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Files uploaded successfully",
    "files": [
      {
        "url": "https://cdn.example.com/ip-assets/file1.png",
        "key": "ip-assets/20240115_abcdef12_file1.png",
        "size": 2048576,
        "mime_type": "image/png",
        "filename": "artwork1.png"
      }
    ]
  }
}
```

### Get Popular IP Assets
Gets list of popular IP assets.

```
GET /ip-assets/popular?limit=10
```

**Response:**
```json
{
  "success": true,
  "data": {
    "ip_assets": [...]
  }
}
```

### Get Featured IP Assets
Gets list of featured IP assets.

```
GET /ip-assets/featured?limit=10
```

## License Endpoints

### Apply for License
Submits an application to license an IP asset.

```
POST /licenses/apply
```
*Requires Authentication (Secondary Creator role)*

**Request Body:**
```json
{
  "ip_asset_id": "123e4567-e89b-12d3-a456-426614174000",
  "license_terms_id": "license-terms-id",
  "application_data": {
    "intended_use": "Creating t-shirts and merchandise",
    "business_type": "E-commerce store",
    "expected_volume": "500-1000 units per month",
    "portfolio_url": "https://mystore.com/portfolio"
  },
  "message": "I would like to use this artwork for my t-shirt business targeting young adults."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "License application submitted successfully",
    "application": {
      "id": "application-id",
      "ip_asset_id": "123e4567-e89b-12d3-a456-426614174000",
      "applicant_id": "applicant-id",
      "status": "pending",
      "application_data": { ... },
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Get License Applications
Retrieves user's license applications.

```
GET /licenses/applications
```
*Requires Authentication*

**Query Parameters:**
- `ip_asset_id`: Filter by IP asset
- `status`: Filter by application status
- `license_type`: Filter by license type
- Standard pagination parameters

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "application-id",
      "ip_asset": {
        "id": "ip-asset-id",
        "title": "Digital Art Collection #1",
        "creator": {
          "username": "artistjohn"
        }
      },
      "license_terms": {
        "license_type": "standard",
        "revenue_share_percentage": 20.0
      },
      "status": "approved",
      "approved_at": "2024-01-16T09:15:00Z",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Get License Application Details
Retrieves detailed information about a license application.

```
GET /licenses/:id
```
*Requires Authentication*

**Response:**
```json
{
  "success": true,
  "data": {
    "application": {
      "id": "application-id",
      "ip_asset": { ... },
      "applicant": { ... },
      "license_terms": { ... },
      "application_data": { ... },
      "status": "approved",
      "approved_at": "2024-01-16T09:15:00Z",
      "approved_by": "creator-id",
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Approve License Application
Approves a license application (IP creator only).

```
PUT /licenses/:id/approve
```
*Requires Authentication (Creator - own IP assets only)*

**Request Body:**
```json
{
  "message": "Application approved. Welcome to use my artwork!"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "License application approved successfully",
    "application": { ... }
  }
}
```

### Reject License Application
Rejects a license application.

```
PUT /licenses/:id/reject
```
*Requires Authentication (Creator - own IP assets only)*

**Request Body:**
```json
{
  "reason": "Portfolio does not meet quality standards",
  "message": "Please improve your portfolio and reapply."
}
```

### Revoke License
Revokes an active license.

```
PUT /licenses/:id/revoke
```
*Requires Authentication (Creator - own IP assets only)*

**Request Body:**
```json
{
  "reason": "Terms of use violation",
  "message": "License revoked due to unauthorized usage."
}
```

### Verify License
Verifies if a license is valid and active.

```
GET /licenses/:id/verify
```

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true,
    "license": {
      "id": "license-id",
      "status": "approved",
      "expires_at": null,
      "ip_asset": { ... },
      "license_terms": { ... }
    }
  }
}
```

### Get My Licenses
Gets user's approved licenses.

```
GET /licenses/my-licenses
```
*Requires Authentication*

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "license-id",
      "ip_asset": { ... },
      "license_terms": { ... },
      "approved_at": "2024-01-16T09:15:00Z",
      "expires_at": null
    }
  ]
}
```

## Product Endpoints

### Get Products
Retrieves list of products with filtering.

```
GET /products
```

**Query Parameters:**
- `category`: Filter by category
- `creator_id`: Filter by creator
- `license_id`: Filter by license
- `price_min`: Minimum price filter
- `price_max`: Maximum price filter
- `tags`: Comma-separated tags
- `in_stock`: Filter by availability
- Standard pagination parameters

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "product-id",
      "title": "Abstract Art T-Shirt",
      "description": "Premium quality t-shirt featuring beautiful abstract artwork",
      "category": "apparel",
      "price": 29.99,
      "inventory_count": 150,
      "images": [
        "https://cdn.example.com/products/tshirt1.jpg",
        "https://cdn.example.com/products/tshirt2.jpg"
      ],
      "specifications": {
        "material": "100% Cotton",
        "sizes": ["S", "M", "L", "XL"],
        "colors": ["White", "Black", "Navy"]
      },
      "status": "active",
      "authenticity_verified": true,
      "tags": ["t-shirt", "abstract", "art", "premium"],
      "sales_count": 45,
      "rating": 4.8,
      "creator": {
        "username": "fashionstore",
        "verification_level": "verified"
      },
      "license": {
        "ip_asset": {
          "title": "Digital Art Collection #1",
          "creator": {
            "username": "artistjohn"
          }
        }
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Create Product
Creates a new product using a licensed IP.

```
POST /products
```
*Requires Authentication (Secondary Creator role)*

**Request Body:**
```json
{
  "license_id": "approved-license-id",
  "title": "Abstract Art T-Shirt",
  "description": "Premium quality t-shirt featuring beautiful abstract artwork",
  "category": "apparel",
  "price": 29.99,
  "inventory_count": 100,
  "images": [
    "https://cdn.example.com/products/tshirt1.jpg"
  ],
  "specifications": {
    "material": "100% Cotton",
    "sizes": ["S", "M", "L", "XL"],
    "care_instructions": "Machine wash cold, tumble dry low"
  },
  "tags": ["t-shirt", "abstract", "art"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Product created successfully",
    "product": {
      "id": "product-id",
      "title": "Abstract Art T-Shirt",
      "status": "draft",
      "authenticity_verified": true,
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Get Product Details
Retrieves detailed product information.

```
GET /products/:id
```

**Response:**
```json
{
  "success": true,
  "data": {
    "product": {
      "id": "product-id",
      "title": "Abstract Art T-Shirt",
      "description": "Premium quality t-shirt featuring beautiful abstract artwork",
      "category": "apparel",
      "price": 29.99,
      "inventory_count": 150,
      "images": [...],
      "specifications": {...},
      "status": "active",
      "authenticity_verified": true,
      "view_count": 1250,
      "sales_count": 45,
      "rating": 4.8,
      "review_count": 12,
      "creator": {...},
      "license": {
        "ip_asset": {...},
        "license_terms": {...}
      },
      "auth_chain": [
        {
          "id": "auth-chain-id",
          "verification_code": "ABC123XYZ789",
          "blockchain_hash": "0x1234567890abcdef...",
          "is_active": true
        }
      ],
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Update Product
Updates product information.

```
PUT /products/:id
```
*Requires Authentication (Creator - own products only)*

**Request Body:**
```json
{
  "title": "Updated Abstract Art T-Shirt",
  "price": 34.99,
  "inventory_count": 200,
  "status": "active"
}
```

### Purchase Product
Initiates product purchase.

```
POST /products/:id/purchase
```
*Requires Authentication (Buyer role)*

**Request Body:**
```json
{
  "quantity": 2,
  "payment_method": "stripe",
  "shipping_info": {
    "name": "John Doe",
    "address": "123 Main St",
    "city": "New York",
    "state": "NY",
    "zip": "10001",
    "country": "US"
  },
  "notes": "Please pack carefully"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Product purchased successfully",
    "transaction": {
      "id": "transaction-id",
      "amount": 59.98,
      "platform_fee": 2.99,
      "revenue_shares": {
        "ip_creator_share": 11.99,
        "secondary_creator_share": 44.99
      },
      "status": "pending",
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Verify Product Authenticity
Verifies that a product is authentic and properly licensed.

```
GET /products/:id/verify
```

**Response:**
```json
{
  "success": true,
  "data": {
    "verified": true,
    "authorization_chain": {
      "id": "auth-chain-id",
      "product_id": "product-id",
      "ip_asset": {
        "title": "Digital Art Collection #1",
        "creator": "artistjohn"
      },
      "license": {
        "status": "approved",
        "approved_at": "2024-01-16T09:15:00Z"
      },
      "verification_code": "ABC123XYZ789",
      "blockchain_hash": "0x1234567890abcdef...",
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Upload Product Images
Uploads images for a product.

```
POST /products/upload-images
```
*Requires Authentication*
*Content-Type: multipart/form-data*

**Request:**
```
Form Data:
images: [multiple image files]
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Images uploaded successfully",
    "images": [
      {
        "url": "https://cdn.example.com/products/img1.jpg",
        "key": "products/20240115_xyz123_img1.jpg",
        "size": 1024000,
        "mime_type": "image/jpeg"
      }
    ]
  }
}
```

## Payment Endpoints

### Create Payment Intent
Creates a payment intent for processing payments.

```
POST /payments/intent
```
*Requires Authentication*

**Request Body:**
```json
{
  "amount": 59.98,
  "currency": "usd",
  "payment_method": "stripe",
  "metadata": {
    "transaction_id": "transaction-id",
    "product_id": "product-id"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "client_secret": "pi_1234567890_secret_abcdef",
    "payment_id": "pi_1234567890",
    "status": "requires_confirmation"
  }
}
```

### Confirm Payment
Confirms a payment after successful processing.

```
POST /payments/confirm
```

**Request Body:**
```json
{
  "payment_intent_id": "pi_1234567890",
  "transaction_id": "transaction-id"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Payment completed successfully"
  }
}
```

### Get Payment History
Retrieves user's payment history.

```
GET /payments/history
```
*Requires Authentication*

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "transaction-id",
      "transaction_type": "product_sale",
      "amount": 59.98,
      "platform_fee": 2.99,
      "payment_method": "stripe",
      "status": "completed",
      "product": {
        "title": "Abstract Art T-Shirt"
      },
      "processed_at": "2024-01-15T10:35:00Z",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Get User Balance
Gets user's current balance and earnings.

```
GET /payments/balance
```
*Requires Authentication*

**Response:**
```json
{
  "success": true,
  "data": {
    "balance": {
      "total_earnings": 1250.50,
      "pending_payouts": 0,
      "available_balance": 1250.50,
      "currency": "USD"
    }
  }
}
```

### Request Payout
Requests a payout of available balance.

```
POST /payments/payout
```
*Requires Authentication*

**Request Body:**
```json
{
  "amount": 500.00,
  "method": "bank_transfer",
  "account_info": {
    "account_number": "****1234",
    "routing_number": "021000021"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Payout request submitted successfully"
  }
}
```

## Verification Endpoints

### Verify Product by Code
Verifies product authenticity using verification code (public endpoint).

```
GET /verify/:code
```

**Example:**
```
GET /verify/ABC123XYZ789
```

**Response:**
```json
{
  "success": true,
  "data": {
    "verified": true,
    "product": {
      "title": "Abstract Art T-Shirt",
      "creator": "fashionstore"
    },
    "ip_asset": {
      "title": "Digital Art Collection #1",
      "creator": "artistjohn"
    },
    "license": {
      "status": "approved",
      "approved_at": "2024-01-16T09:15:00Z"
    },
    "authorization_chain": {
      "verification_code": "ABC123XYZ789",
      "blockchain_hash": "0x1234567890abcdef...",
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### Verify Authorization Chain
Verifies an authorization chain by ID.

```
GET /verify/chain/:id
```

**Response:**
```json
{
  "success": true,
  "data": {
    "valid": true
  }
}
```

### Get Authorization Chain History
Gets the complete authorization history for a product.

```
GET /verify/chain/:product_id/history
```

**Response:**
```json
{
  "success": true,
  "data": {
    "authorization_chains": [
      {
        "id": "auth-chain-id",
        "verification_code": "ABC123XYZ789",
        "blockchain_hash": "0x1234567890abcdef...",
        "is_active": true,
        "created_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

## Admin Endpoints

*All admin endpoints require Admin authentication*

### Get Dashboard Statistics
Retrieves platform statistics for admin dashboard.

```
GET /admin/dashboard/stats
```

**Response:**
```json
{
  "success": true,
  "data": {
    "stats": {
      "total_users": 15420,
      "active_users": 12350,
      "new_users_this_month": 1250,
      "total_revenue": 125000.50,
      "monthly_revenue": 15000.75,
      "total_ips": 3240,
      "pending_ip_verification": 45,
      "total_products": 8960,
      "active_licenses": 5670,
      "pending_licenses": 23,
      "total_transactions": 45230,
      "user_growth": 12.5,
      "revenue_growth": 8.3
    }
  }
}
```

### Get Users
Retrieves list of users with admin filters.

```
GET /admin/users
```

**Query Parameters:**
- `user_type`: Filter by user type
- `status`: Filter by user status
- `verification_level`: Filter by verification level
- `created_after`: Filter by creation date
- `created_before`: Filter by creation date
- Standard pagination and search parameters

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "user-id",
      "username": "johndoe",
      "email": "john@example.com",
      "user_type": "creator",
      "status": "active",
      "verification_level": "verified",
      "created_at": "2024-01-15T10:30:00Z",
      "last_login_at": "2024-01-20T15:45:00Z",
      "total_revenue": 2500.00,
      "total_ips": 15,
      "total_products": 0
    }
  ]
}
```

### Update User Status
Updates a user's account status.

```
PUT /admin/users/:id/status
```

**Request Body:**
```json
{
  "status": "suspended",
  "reason": "Terms of service violation"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "User has been suspended"
  }
}
```

### Update User Verification Level
Updates a user's verification level.

```
PUT /admin/users/:id/verify
```

**Request Body:**
```json
{
  "verification_level": "verified"
}
```

### Get Pending IP Assets
Gets IP assets pending verification.

```
GET /admin/ip-assets/pending
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "ip-asset-id",
      "title": "Digital Art Collection #1",
      "creator": {
        "username": "artistjohn",
        "verification_level": "verified"
      },
      "category": "art",
      "verification_status": "pending",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Approve IP Asset
Approves an IP asset for use on the platform.

```
PUT /admin/ip-assets/:id/approve
```

**Request Body:**
```json
{
  "message": "Great artwork! Approved for platform use."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "IP asset has been approved"
  }
}
```

### Reject IP Asset
Rejects an IP asset.

```
PUT /admin/ip-assets/:id/reject
```

**Request Body:**
```json
{
  "reason": "Copyright concerns",
  "message": "Please provide proof of ownership before resubmitting."
}
```

### Get Transactions
Gets platform transaction history.

```
GET /admin/transactions
```

**Query Parameters:**
- `transaction_type`: Filter by transaction type
- `status`: Filter by transaction status
- `buyer_id`: Filter by buyer
- `seller_id`: Filter by seller
- `amount_min`: Minimum amount filter
- `amount_max`: Maximum amount filter
- Date range filters

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "transaction-id",
      "transaction_type": "product_sale",
      "amount": 59.98,
      "platform_fee": 2.99,
      "status": "completed",
      "buyer": {
        "username": "buyer123"
      },
      "seller": {
        "username": "seller456"
      },
      "product": {
        "title": "Abstract Art T-Shirt"
      },
      "created_at": "2024-01-15T10:30:00Z",
      "processed_at": "2024-01-15T10:35:00Z"
    }
  ]
}
```

### Process Refund
Processes a refund for a transaction.

```
POST /admin/transactions/:id/refund
```

**Request Body:**
```json
{
  "reason": "Product defect reported by customer"
}
```

### Get Analytics
Gets platform analytics data.

```
GET /admin/analytics
```

**Query Parameters:**
- `start_date`: Start date (YYYY-MM-DD)
- `end_date`: End date (YYYY-MM-DD)
- `metrics`: Comma-separated list of metrics

**Example:**
```
GET /admin/analytics?start_date=2024-01-01&end_date=2024-01-31&metrics=user_registrations,revenue
```

**Response:**
```json
{
  "success": true,
  "data": {
    "analytics": {
      "user_registrations": 1250,
      "ip_creations": 340,
      "license_applications": 890,
      "product_sales": 2340,
      "revenue": 45000.50
    },
    "start_date": "2024-01-01",
    "end_date": "2024-01-31",
    "metrics": ["user_registrations", "revenue"]
  }
}
```

### Get Platform Settings
Gets current platform settings.

```
GET /admin/settings
```

**Response:**
```json
{
  "success": true,
  "data": {
    "settings": {
      "general.platform_name": {
        "value": "IP Marketplace",
        "data_type": "string"
      },
      "payments.platform_fee_percentage": {
        "value": 5.0,
        "data_type": "float"
      },
      "verification.auto_verify_creators": {
        "value": false,
        "data_type": "boolean"
      }
    }
  }
}
```

### Update Platform Settings
Updates platform settings.

```
PUT /admin/settings
```

**Request Body:**
```json
{
  "payments.platform_fee_percentage": 5.5,
  "verification.auto_verify_creators": true,
  "general.maintenance_mode": false
}
```

## Webhook Events

The platform can send webhook notifications for various events.

### Webhook Configuration
Configure webhook endpoints in admin settings:
```json
{
  "webhook_url": "https://your-app.com/webhooks/ip-marketplace",
  "webhook_secret": "your-webhook-secret",
  "events": ["user.registered", "license.approved", "product.purchased"]
}
```

### Event Types
- `user.registered` - New user registration
- `user.verified` - User verification status changed
- `ip_asset.created` - New IP asset created
- `ip_asset.approved` - IP asset approved
- `license.applied` - License application submitted
- `license.approved` - License application approved
- `product.created` - New product created
- `product.purchased` - Product purchased
- `payment.completed` - Payment processed successfully

### Webhook Payload Example
```json
{
  "event": "product.purchased",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "transaction": {
      "id": "transaction-id",
      "amount": 59.98,
      "product": {
        "title": "Abstract Art T-Shirt"
      },
      "buyer": {
        "username": "buyer123"
      }
    }
  }
}
```

## SDKs and Examples

### JavaScript/TypeScript SDK
```javascript
import { IPMarketplaceClient } from '@ip-marketplace/sdk';

const client = new IPMarketplaceClient({
  baseURL: 'https://api.ipmarketplace.com/v1',
  apiKey: 'your-api-key'
});

// Get IP assets
const ipAssets = await client.ipAssets.list({
  category: 'art',
  limit: 10
});

// Create product
const product = await client.products.create({
  licenseId: 'license-id',
  title: 'My Product',
  price: 29.99
});
```

### Python SDK
```python
from ip_marketplace import IPMarketplaceClient

client = IPMarketplaceClient(
    base_url='https://api.ipmarketplace.com/v1',
    api_key='your-api-key'
)

# Get IP assets
ip_assets = client.ip_assets.list(category='art', limit=10)

# Create product
product = client.products.create({
    'license_id': 'license-id',
    'title': 'My Product',
    'price': 29.99
})
```

### cURL Examples
```bash
# Login
curl -X POST https://api.ipmarketplace.com/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}'

# Get IP assets
curl -X GET "https://api.ipmarketplace.com/v1/ip-assets?category=art&limit=10" \
  -H "Authorization: Bearer YOUR_TOKEN"

# Create product
curl -X POST https://api.ipmarketplace.com/v1/products \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "license_id": "license-id",
    "title": "My Product",
    "price": 29.99
  }'
```

## Support

For API support:
- **Documentation**: [https://docs.ipmarketplace.com](https://docs.ipmarketplace.com)
- **Support Email**: api-support@ipmarketplace.com
- **Developer Discord**: [https://discord.gg/ipmarketplace](https://discord.gg/ipmarketplace)
- **Status Page**: [https://status.ipmarketplace.com](https://status.ipmarketplace.com)

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for API version history and breaking changes.