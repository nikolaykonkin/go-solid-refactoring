package main

import (
	"errors"
	"testing"
)

// MockWriter — мок RepositoryWriter: не пишет в реальную БД, а сохраняет заказы в памяти
// Поле err позволяет задать ошибку, которую вернет SaveOrder,
// calls — счетчик вызовов для проверки контракта OrderService
type MockWriter struct {
	saved []Order
	err   error
	calls int
}

func (m *MockWriter) SaveOrder(order Order) error {
	m.calls++
	if m.err != nil {
		return m.err
	}
	m.saved = append(m.saved, order)
	return nil
}

// MockNotifier — мок Notifier: фиксирует получателя и текст сообщения, без реальной отправки email/sms
// Поле err позволяет задать ошибку, которую вернет Send, calls — счетчик вызовов
type MockNotifier struct {
	sentTo      []string
	sentMessage []string
	err         error
	calls       int
}

func (m *MockNotifier) Send(customer string, message string) error {
	m.calls++
	if m.err != nil {
		return m.err
	}
	m.sentTo = append(m.sentTo, customer)
	m.sentMessage = append(m.sentMessage, message)
	return nil
}

func TestOrderService_CreateOrder_Success(t *testing.T) {
	writer := &MockWriter{}
	notifier := &MockNotifier{}
	service := NewOrderService(writer, notifier)

	err := service.CreateOrder("Тест", []string{"item1"}, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writer.saved) != 1 {
		t.Fatalf("ожидался 1 сохранённый заказ, получено %d", len(writer.saved))
	}

	got := writer.saved[0]
	want := Order{
		Customer: "Тест",
		Products: "[item1]",
		Total:    100,
		Status:   "pending",
	}
	if got.Customer != want.Customer || got.Products != want.Products ||
		got.Total != want.Total || got.Status != want.Status {
		t.Errorf("сохранённый заказ = %+v, ожидался %+v", got, want)
	}

	if len(notifier.sentTo) != 1 || notifier.sentTo[0] != "Тест" {
		t.Fatalf("ожидалось уведомление клиенту Тест, получено %v", notifier.sentTo)
	}

	wantMessage := "Ваш заказ на сумму 100.00 принят в обработку"
	if len(notifier.sentMessage) != 1 || notifier.sentMessage[0] != wantMessage {
		t.Errorf("текст уведомления = %q, ожидался %q", notifier.sentMessage, wantMessage)
	}
}

func TestOrderService_CreateOrder_RepositoryError(t *testing.T) {
	repoErr := errors.New("ошибка записи в БД")
	writer := &MockWriter{err: repoErr}
	notifier := &MockNotifier{}
	service := NewOrderService(writer, notifier)

	err := service.CreateOrder("Тест", []string{"item1"}, 100)
	if err == nil {
		t.Fatal("ожидалась ошибка, получен nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("ошибка = %v, должна оборачивать %v", err, repoErr)
	}

	if writer.calls != 1 {
		t.Errorf("ожидался 1 вызов SaveOrder, получено %d", writer.calls)
	}
	if len(writer.saved) != 0 {
		t.Errorf("при ошибке записи заказ не должен считаться сохранённым, получено %d", len(writer.saved))
	}

	if notifier.calls != 0 {
		t.Errorf("Notifier.Send не должен вызываться при ошибке записи, вызовов: %d", notifier.calls)
	}
}

func TestOrderService_CreateOrder_NotifierError(t *testing.T) {
	sendErr := errors.New("ошибка отправки уведомления")
	writer := &MockWriter{}
	notifier := &MockNotifier{err: sendErr}
	service := NewOrderService(writer, notifier)

	err := service.CreateOrder("Тест", []string{"item1"}, 100)
	if err == nil {
		t.Fatal("ожидалась ошибка, получен nil")
	}
	if !errors.Is(err, sendErr) {
		t.Errorf("ошибка = %v, должна оборачивать %v", err, sendErr)
	}

	if len(writer.saved) != 1 {
		t.Errorf("заказ должен быть сохранён до отправки уведомления, сохранено %d", len(writer.saved))
	}

	if notifier.calls != 1 {
		t.Errorf("ожидался 1 вызов Send, получено %d", notifier.calls)
	}
}
