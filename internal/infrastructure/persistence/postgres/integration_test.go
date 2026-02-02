package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"turtle/internal/domain"
	"turtle/internal/domain/aggregates"
	"turtle/internal/domain/valueobjects"
	infraPostgres "turtle/internal/infrastructure/persistence/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================================
// TEST SETUP
// ============================================================================

func setupTestDB(t *testing.T) *gorm.DB {
	// Use test database
	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=turtle_delivery_test sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to test database")

	// Auto-migrate tables
	err = db.AutoMigrate(
		&infraPostgres.UserModel{},
		&infraPostgres.CaptainProfileModel{},
		&infraPostgres.AdminProfileModel{},
		&infraPostgres.AddressModel{},
		&infraPostgres.OTPSessionModel{},
		&infraPostgres.RefreshTokenModel{},
	)
	require.NoError(t, err, "Failed to migrate test database")

	// Clean tables before tests
	db.Exec("TRUNCATE TABLE users CASCADE")
	db.Exec("TRUNCATE TABLE addresses CASCADE")
	db.Exec("TRUNCATE TABLE otp_sessions CASCADE")
	db.Exec("TRUNCATE TABLE refresh_tokens CASCADE")

	return db
}

func cleanupTestDB(t *testing.T, db *gorm.DB) {
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()
}

// ============================================================================
// USER REPOSITORY TESTS
// ============================================================================

func TestUserRepository_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create customer
	email := "john@example.com"
	user, err := aggregates.NewUser(
		"",
		"John",
		"Doe",
		&email,
		nil,
		aggregates.RoleCustomer,
		"GOOGLE",
	)
	require.NoError(t, err)

	// Save user
	err = repo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID())

	// Find by ID
	found, err := repo.FindByID(ctx, user.ID())
	require.NoError(t, err)
	assert.Equal(t, "John", found.FirstName())
	assert.Equal(t, "Doe", found.LastName())
	assert.Equal(t, email, *found.Email())

	// Find by email
	foundByEmail, err := repo.FindByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, user.ID(), foundByEmail.ID())
}

func TestUserRepository_OptimisticLocking(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user
	email := "test@example.com"
	user, _ := aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	repo.Create(ctx, user)

	// Load user twice (simulating two concurrent requests)
	user1, _ := repo.FindByID(ctx, user.ID())
	user2, _ := repo.FindByID(ctx, user.ID())

	// First update succeeds
	user1.UpdateProfile("Updated1", "Name1", "")
	err1 := repo.Update(ctx, user1)
	require.NoError(t, err1)

	// Second update should fail (version mismatch)
	user2.UpdateProfile("Updated2", "Name2", "")
	err2 := repo.Update(ctx, user2)
	assert.Equal(t, domain.ErrConcurrentModification, err2)
}

func TestUserRepository_WalletOperations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user
	email := "wallet@example.com"
	user, _ := aggregates.NewUser("", "Wallet", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	repo.Create(ctx, user)

	// Add to wallet
	amount, _ := valueobjects.FromMajorUnit(100, "INR")
	err := user.AddToWallet(amount)
	require.NoError(t, err)

	// Update wallet with optimistic locking
	err = repo.UpdateWallet(ctx, user.ID(), user.WalletBalance().Amount(), 1)
	require.NoError(t, err)

	// Load user and verify balance
	loaded, _ := repo.FindByID(ctx, user.ID())
	assert.Equal(t, int64(10000), loaded.WalletBalance().Amount()) // 100 * 100 paise
}

func TestUserRepository_CaptainSearch(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create captains at different locations
	for i := 0; i < 5; i++ {
		phone := fmt.Sprintf("+9198765432%d%d", i, i)
		captain, _ := aggregates.NewUser("", "Captain", fmt.Sprintf("%d", i), nil, &phone, aggregates.RoleCaptain, "PHONE")
		captain.VerifyPhone()

		// Approve KYC
		docs := map[string]string{"LICENSE": "url", "VEHICLE_RC": "url", "PROFILE_PHOTO": "url"}
		captain.SubmitKYCDocuments(docs)
		captain.ApproveKyc()

		// Go online at different locations
		lat := 17.385 + (float64(i) * 0.01)
		lng := 78.486 + (float64(i) * 0.01)
		location, _ := valueobjects.NewLocation(lat, lng)
		captain.GoOnline(location)

		repo.Create(ctx, captain)
	}

	// Search for captains near center point
	captains, err := repo.FindCaptainsNearby(ctx, 17.385, 78.486, 5.0)
	require.NoError(t, err)
	assert.Greater(t, len(captains), 0)

	t.Logf("Found %d captains within 5km", len(captains))
}

// ============================================================================
// ADDRESS REPOSITORY TESTS
// ============================================================================

func TestAddressRepository_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	addressRepo := infraPostgres.NewAddressRepository(db)
	userRepo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user first
	email := "address@example.com"
	user, _ := aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	userRepo.Create(ctx, user)

	// Create address
	location, _ := valueobjects.NewLocation(17.385, 78.486)
	address, err := aggregates.NewAddress(
		"",
		user.ID(),
		aggregates.AddressLabelHome,
		"123 Main St",
		"Apt 4B",
		"Near Park",
		"Hyderabad",
		"Telangana",
		"India",
		"500001",
		location,
		"John Doe",
		"+919876543210",
	)
	require.NoError(t, err)

	// Save address
	err = addressRepo.Create(ctx, address)
	require.NoError(t, err)

	// Find addresses by user
	addresses, err := addressRepo.FindByUserID(ctx, user.ID())
	require.NoError(t, err)
	assert.Len(t, addresses, 1)
	assert.Equal(t, "Hyderabad", addresses[0].City())
}

func TestAddressRepository_SmartSuggestions(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	addressRepo := infraPostgres.NewAddressRepository(db)
	userRepo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user
	email := "suggestions@example.com"
	user, _ := aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	userRepo.Create(ctx, user)

	// Create address
	location, _ := valueobjects.NewLocation(17.385, 78.486)
	address, _ := aggregates.NewAddress(
		"",
		user.ID(),
		aggregates.AddressLabelHome,
		"123 Main St",
		"",
		"",
		"Hyderabad",
		"Telangana",
		"India",
		"500001",
		location,
		"",
		"",
	)
	addressRepo.Create(ctx, address)

	// Increment usage multiple times to build pattern
	for i := 0; i < 5; i++ {
		err := addressRepo.IncrementUsage(ctx, address.ID())
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond) // Small delay
	}

	// Get suggestions
	suggestions, err := addressRepo.FindSuggestedAddresses(ctx, user.ID(), 3)
	require.NoError(t, err)
	assert.Greater(t, len(suggestions), 0)

	t.Logf("Found %d suggested addresses", len(suggestions))
}

func TestAddressRepository_UsageAnalytics(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	addressRepo := infraPostgres.NewAddressRepository(db)
	userRepo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user and address
	email := "analytics@example.com"
	user, _ := aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	userRepo.Create(ctx, user)

	location, _ := valueobjects.NewLocation(17.385, 78.486)
	address, _ := aggregates.NewAddress(
		"",
		user.ID(),
		aggregates.AddressLabelWork,
		"Office Address",
		"",
		"",
		"Hyderabad",
		"Telangana",
		"India",
		"500001",
		location,
		"",
		"",
	)
	addressRepo.Create(ctx, address)

	// Simulate usage
	for i := 0; i < 10; i++ {
		addressRepo.IncrementUsage(ctx, address.ID())
		time.Sleep(50 * time.Millisecond)
	}

	// Get usage patterns
	patterns, err := addressRepo.GetUsagePatterns(ctx, user.ID())
	require.NoError(t, err)
	assert.Greater(t, len(patterns), 0)

	for _, pattern := range patterns {
		t.Logf("Address: %s, Total: %d, Morning: %.1f%%, Weekday: %.1f%%",
			pattern.AddressID,
			pattern.TotalUsage,
			pattern.MorningPercentage,
			pattern.WeekdayPercentage,
		)
	}
}

// ============================================================================
// OTP REPOSITORY TESTS
// ============================================================================

func TestOTPRepository_CreateAndVerify(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := infraPostgres.NewOTPRepository(db)
	ctx := context.Background()

	// Create OTP
	phone := "+919876543210"
	code := "123456"
	expiresAt := time.Now().Add(15 * time.Minute)

	err := repo.Create(ctx, phone, code, "LOGIN", expiresAt)
	require.NoError(t, err)

	// Find OTP
	session, err := repo.FindByTarget(ctx, phone, "LOGIN")
	require.NoError(t, err)
	assert.Equal(t, code, session.Code)
	assert.False(t, session.Used)
	assert.Equal(t, 0, session.Attempts)

	// Verify OTP
	valid, err := repo.IsOTPValid(ctx, phone, "LOGIN", code)
	require.NoError(t, err)
	assert.True(t, valid)

	// Mark as used
	err = repo.MarkAsUsed(ctx, phone, "LOGIN")
	require.NoError(t, err)

	// Verify it's now used
	session, _ = repo.FindByTarget(ctx, phone, "LOGIN")
	assert.True(t, session.Used)
}

func TestOTPRepository_RateLimiting(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	repo := infraPostgres.NewOTPRepository(db)
	ctx := context.Background()

	phone := "+919876543210"

	// Create multiple OTP sessions
	for i := 0; i < 5; i++ {
		err := repo.Create(ctx, phone, "123456", "LOGIN", time.Now().Add(15*time.Minute))
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Count recent requests
	count, err := repo.CountRecentOTPRequests(ctx, phone, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

// ============================================================================
// REFRESH TOKEN REPOSITORY TESTS
// ============================================================================

func TestRefreshTokenRepository_CreateAndFind(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tokenRepo := infraPostgres.NewRefreshTokenRepository(db)
	userRepo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user
	email := "token@example.com"
	user, _ := aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	userRepo.Create(ctx, user)

	// Create refresh token
	token := "refresh_token_123456789"
	expiresAt := time.Now().Add(60 * 24 * time.Hour)

	err := tokenRepo.Create(ctx, user.ID(), token, "IOS", expiresAt)
	require.NoError(t, err)

	// Find token
	found, err := tokenRepo.FindByToken(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, user.ID(), found.UserID)
	assert.False(t, found.IsRevoked)

	// Revoke token
	err = tokenRepo.Revoke(ctx, token)
	require.NoError(t, err)

	// Verify revoked
	found, _ = tokenRepo.FindByToken(ctx, token)
	assert.True(t, found.IsRevoked)
}

func TestRefreshTokenRepository_SessionManagement(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tokenRepo := infraPostgres.NewRefreshTokenRepository(db)
	userRepo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user
	email := "sessions@example.com"
	user, _ := aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	userRepo.Create(ctx, user)

	// Create multiple tokens for different devices
	devices := []string{"IOS", "ANDROID", "WEB"}
	for i, device := range devices {
		token := fmt.Sprintf("token_%s_%d", device, i)
		expiresAt := time.Now().Add(60 * 24 * time.Hour)
		tokenRepo.Create(ctx, user.ID(), token, device, expiresAt)
	}

	// Get active sessions
	sessions, err := tokenRepo.GetActiveSessions(ctx, user.ID())
	require.NoError(t, err)
	assert.Len(t, sessions, 3)

	for _, session := range sessions {
		t.Logf("Device: %s, Tokens: %d", session.Device, session.TokenCount)
	}

	// Revoke all for one device
	err = tokenRepo.RevokeAllForDevice(ctx, user.ID(), "IOS")
	require.NoError(t, err)

	// Verify
	count, _ := tokenRepo.CountActiveTokens(ctx, user.ID())
	assert.Equal(t, int64(2), count) // Only ANDROID and WEB remain
}

// ============================================================================
// CLEANUP TESTS
// ============================================================================

func TestCleanupOperations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	otpRepo := infraPostgres.NewOTPRepository(db)
	tokenRepo := infraPostgres.NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create expired OTP
	phone := "+919876543210"
	expiredOTP := time.Now().Add(-1 * time.Hour)
	otpRepo.Create(ctx, phone, "123456", "LOGIN", expiredOTP)

	// Cleanup expired OTPs
	err := otpRepo.DeleteExpired(ctx)
	require.NoError(t, err)

	// Verify deleted
	_, err = otpRepo.FindByTarget(ctx, phone, "LOGIN")
	assert.Equal(t, domain.ErrNotFound, err)

	// Create expired token
	email := "cleanup@example.com"
	aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	// (Assuming user repo is available)

	expiredToken := "expired_token"
	expiredAt := time.Now().Add(-1 * time.Hour)
	tokenRepo.Create(ctx, "user-id", expiredToken, "IOS", expiredAt)

	// Cleanup expired tokens
	err = tokenRepo.DeleteExpired(ctx)
	require.NoError(t, err)

	t.Log("Cleanup operations completed successfully")
}

// ============================================================================
// BENCHMARK TESTS
// ============================================================================

func BenchmarkUserRepository_FindByID(b *testing.B) {
	db := setupTestDB(&testing.T{})
	repo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	email := "benchmark@example.com"
	user, _ := aggregates.NewUser("", "Benchmark", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	repo.Create(ctx, user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.FindByID(ctx, user.ID())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAddressRepository_IncrementUsage(b *testing.B) {
	db := setupTestDB(&testing.T{})
	addressRepo := infraPostgres.NewAddressRepository(db)
	userRepo := infraPostgres.NewUserRepository(db)
	ctx := context.Background()

	// Create user and address
	email := "benchmark@example.com"
	user, _ := aggregates.NewUser("", "Test", "User", &email, nil, aggregates.RoleCustomer, "GOOGLE")
	userRepo.Create(ctx, user)

	location, _ := valueobjects.NewLocation(17.385, 78.486)
	address, _ := aggregates.NewAddress("", user.ID(), aggregates.AddressLabelHome, "Test St", "", "", "City", "State", "Country", "", location, "", "")
	addressRepo.Create(ctx, address)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := addressRepo.IncrementUsage(ctx, address.ID())
		if err != nil {
			b.Fatal(err)
		}
	}
}
