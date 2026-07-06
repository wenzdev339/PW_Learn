package seed

import (
	"backend/internal/models"
	"backend/internal/password"

	"gorm.io/gorm"
)

// CleanDatabase truncates every table so tests/reset start from a blank
// slate. CASCADE handles foreign-key ordering.
func CleanDatabase(db *gorm.DB) error {
	return db.Exec(`TRUNCATE TABLE
		payments, order_items, orders, reviews, cart_items, carts, addresses, products, categories, users
		RESTART IDENTITY CASCADE`).Error
}

type categorySeed struct {
	Name string
	Slug string
}

var categorySeeds = []categorySeed{
	{Name: "Electronics", Slug: "electronics"},
	{Name: "Home & Kitchen", Slug: "home-kitchen"},
	{Name: "Books", Slug: "books"},
	{Name: "Sports", Slug: "sports"},
}

type productSeed struct {
	Name         string
	Slug         string
	Description  string
	Price        int
	Stock        int
	CategorySlug string
}

var productSeeds = []productSeed{
	{Name: "Wireless Mouse", Slug: "wireless-mouse", Description: "Ergonomic wireless mouse", Price: 59900, Stock: 25, CategorySlug: "electronics"},
	{Name: "Mechanical Keyboard", Slug: "mechanical-keyboard", Description: "RGB mechanical keyboard", Price: 189900, Stock: 10, CategorySlug: "electronics"},
	{Name: "4K Monitor", Slug: "4k-monitor", Description: "27-inch 4K monitor", Price: 899900, Stock: 5, CategorySlug: "electronics"},
	{Name: "USB-C Hub", Slug: "usb-c-hub", Description: "7-in-1 USB-C hub", Price: 129900, Stock: 0, CategorySlug: "electronics"},
	{Name: "Bluetooth Speaker", Slug: "bluetooth-speaker", Description: "Portable Bluetooth speaker", Price: 149900, Stock: 16, CategorySlug: "electronics"},
	{Name: "Non-stick Frying Pan", Slug: "non-stick-frying-pan", Description: "28cm non-stick pan", Price: 45900, Stock: 30, CategorySlug: "home-kitchen"},
	{Name: "Electric Kettle", Slug: "electric-kettle", Description: "1.7L electric kettle", Price: 69900, Stock: 18, CategorySlug: "home-kitchen"},
	{Name: "Coffee Grinder", Slug: "coffee-grinder", Description: "Burr coffee grinder", Price: 159900, Stock: 12, CategorySlug: "home-kitchen"},
	{Name: "Bamboo Cutting Board", Slug: "bamboo-cutting-board", Description: "Set of 3 boards", Price: 29900, Stock: 40, CategorySlug: "home-kitchen"},
	{Name: "Desk Lamp", Slug: "desk-lamp", Description: "LED desk lamp with USB port", Price: 49900, Stock: 28, CategorySlug: "home-kitchen"},
	{Name: "Clean Code", Slug: "clean-code", Description: "A Handbook of Agile Software Craftsmanship", Price: 89900, Stock: 20, CategorySlug: "books"},
	{Name: "The Pragmatic Programmer", Slug: "pragmatic-programmer", Description: "20th Anniversary Edition", Price: 99900, Stock: 15, CategorySlug: "books"},
	{Name: "Designing Data-Intensive Applications", Slug: "ddia", Description: "By Martin Kleppmann", Price: 129900, Stock: 8, CategorySlug: "books"},
	{Name: "Atomic Habits", Slug: "atomic-habits", Description: "By James Clear", Price: 59900, Stock: 50, CategorySlug: "books"},
	{Name: "Notebook Set", Slug: "notebook-set", Description: "Set of 3 hardcover notebooks", Price: 19900, Stock: 60, CategorySlug: "books"},
	{Name: "Yoga Mat", Slug: "yoga-mat", Description: "Non-slip 6mm yoga mat", Price: 39900, Stock: 22, CategorySlug: "sports"},
	{Name: "Adjustable Dumbbells", Slug: "adjustable-dumbbells", Description: "2x 20kg adjustable dumbbells", Price: 349900, Stock: 6, CategorySlug: "sports"},
	{Name: "Running Shoes", Slug: "running-shoes", Description: "Lightweight running shoes", Price: 219900, Stock: 14, CategorySlug: "sports"},
	{Name: "Resistance Bands Set", Slug: "resistance-bands", Description: "5-level resistance bands", Price: 24900, Stock: 35, CategorySlug: "sports"},
	{Name: "Water Bottle", Slug: "water-bottle", Description: "1L insulated water bottle", Price: 34900, Stock: 45, CategorySlug: "sports"},
}

// Run truncates the database and inserts deterministic seed data: 4
// categories, 20 products (at least one with Stock == 0), and the two
// fixed accounts admin@example.com / customer@example.com.
func Run(db *gorm.DB) error {
	if err := CleanDatabase(db); err != nil {
		return err
	}

	categoryBySlug := make(map[string]models.Category, len(categorySeeds))
	for _, c := range categorySeeds {
		category := models.Category{Name: c.Name, Slug: c.Slug}
		if err := db.Create(&category).Error; err != nil {
			return err
		}
		categoryBySlug[c.Slug] = category
	}

	for _, p := range productSeeds {
		category, ok := categoryBySlug[p.CategorySlug]
		if !ok {
			continue
		}
		product := models.Product{
			Name:        p.Name,
			Slug:        p.Slug,
			Description: p.Description,
			Price:       p.Price,
			Stock:       p.Stock,
			Images:      []string{"https://picsum.photos/seed/" + p.Slug + "/600/600"},
			CategoryID:  category.ID,
		}
		if err := db.Create(&product).Error; err != nil {
			return err
		}
	}

	adminHash, err := password.Hash("Admin123!")
	if err != nil {
		return err
	}
	admin := models.User{Email: "admin@example.com", PasswordHash: adminHash, Name: "Admin User", Role: models.RoleAdmin}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}

	customerHash, err := password.Hash("Customer123!")
	if err != nil {
		return err
	}
	customer := models.User{Email: "customer@example.com", PasswordHash: customerHash, Name: "Test Customer", Role: models.RoleCustomer}
	if err := db.Create(&customer).Error; err != nil {
		return err
	}

	cart := models.Cart{UserID: customer.ID}
	return db.Create(&cart).Error
}
