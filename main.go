package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// Order — доменная модель заказа
type Order struct {
	ID       int
	Customer string
	Products string
	Total    float64
	Status   string
}

// RepositoryWriter описывает контракт для сохранения заказа в хранилище
// Любая БД (sqlite, postgres, in-memory и т.д.), реализующая этот метод,
// может быть подставлена в OrderService без изменения его кода (DIP, OCP)
type RepositoryWriter interface {
	SaveOrder(order Order) error
}

// RepositoryInitializer вынесен в отдельный интерфейс (ISP), так как не всем клиентам RepositoryWriter
// требуется инициализация схемы — например, моку в тестах или in-memory реализации она не нужна
type RepositoryInitializer interface {
	Init() error
}

// Notifier описывает контракт отправки уведомления клиенту
// EmailSender и SMSSender реализуют его независимо друг от друга
// и взаимозаменяемы (LSP) — OrderService работает с любым Notifier
type Notifier interface {
	Send(customer string, message string) error
}

// SQLiteRepo — реализация RepositoryWriter и RepositoryInitializer для sqlite
type SQLiteRepo struct {
	db *sql.DB
}

func NewSQLiteRepo(db *sql.DB) *SQLiteRepo {
	return &SQLiteRepo{db: db}
}

func (r *SQLiteRepo) Init() error {
	_, err := r.db.Exec(`
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer TEXT NOT NULL,
		products TEXT NOT NULL,
		total REAL NOT NULL,
		status TEXT NOT NULL
	)`)
	return err
}

func (r *SQLiteRepo) SaveOrder(order Order) error {
	_, err := r.db.Exec(
		"INSERT INTO orders (customer, products, total, status) VALUES (?, ?, ?, ?)",
		order.Customer, order.Products, order.Total, order.Status,
	)
	return err
}

// EmailSender — реализация Notifier через e-mail
type EmailSender struct{}

func NewEmailSender() *EmailSender {
	return &EmailSender{}
}

func (s *EmailSender) Send(customer string, message string) error {
	fmt.Printf("[Email] Уведомление отправлено клиенту %s: %s\n", customer, message)
	return nil
}

// SMSSender — реализация Notifier через SMS
// Добавлена без изменения существующих типов (Open/Closed Principle)
type SMSSender struct{}

func NewSMSSender() *SMSSender {
	return &SMSSender{}
}

func (s *SMSSender) Send(customer string, message string) error {
	fmt.Printf("[SMS] Уведомление отправлено клиенту %s: %s\n", customer, message)
	return nil
}

// OrderService содержит бизнес-логику создания заказа
// Зависит только от абстракций RepositoryWriter и Notifier (DIP), не знает о sqlite, email или sms —
// за счет этого он не требует изменений при добавлении новых баз данных или каналов уведомлений (OCP)
type OrderService struct {
	repo     RepositoryWriter
	notifier Notifier
}

func NewOrderService(repo RepositoryWriter, notifier Notifier) *OrderService {
	return &OrderService{repo: repo, notifier: notifier}
}

func (s *OrderService) CreateOrder(customer string, products []string, total float64) error {
	order := Order{
		Customer: customer,
		Products: fmt.Sprintf("%v", products),
		Total:    total,
		Status:   "pending",
	}

	if err := s.repo.SaveOrder(order); err != nil {
		return fmt.Errorf("сохранение заказа: %w", err)
	}

	message := fmt.Sprintf("Ваш заказ на сумму %.2f принят в обработку", total)
	if err := s.notifier.Send(customer, message); err != nil {
		return fmt.Errorf("отправка уведомления: %w", err)
	}

	return nil
}

func main() {
	db, err := sql.Open("sqlite3", "orders.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := NewSQLiteRepo(db)
	if err := repo.Init(); err != nil {
		log.Fatal(err)
	}

	// Демонстрация: OrderService с email-отправителем
	emailService := NewOrderService(repo, NewEmailSender())
	if err := emailService.CreateOrder("Иван", []string{"apple", "banana"}, 10.5); err != nil {
		log.Fatal(err)
	}

	// Демонстрация: тот же OrderService, но с SMS-отправителем
	// Код OrderService при этом не меняется — только внедряемая зависимость
	smsService := NewOrderService(repo, NewSMSSender())
	if err := smsService.CreateOrder("Мария", []string{"orange"}, 5.0); err != nil {
		log.Fatal(err)
	}
}
