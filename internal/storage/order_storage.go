package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"did_helper/internal/models"
)

// OrderStorageManager manages order storage operations
type OrderStorageManager struct {
	baseDir string
}

// NewOrderStorageManager creates a new order storage manager
func NewOrderStorageManager() (*OrderStorageManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".did_helper")
	
	// Create base directory if not exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &OrderStorageManager{
		baseDir: baseDir,
	}, nil
}

// getDIDDirectoryName converts DID to directory name (replace : with -)
func (m *OrderStorageManager) getDIDDirectoryName(did string) string {
	return strings.ReplaceAll(did, ":", "-")
}

// getOrderFilePath returns the file path for orders of a specific DID
func (m *OrderStorageManager) getOrderFilePath(did string) string {
	didDir := m.getDIDDirectoryName(did)
	return filepath.Join(m.baseDir, didDir, "orders.json")
}

// SaveOrder saves or updates an order for a DID
func (m *OrderStorageManager) SaveOrder(did string, order *models.OrderResponse) error {
	orders, err := m.LoadOrders(did)
	if err != nil {
		// If file doesn't exist, start with empty list
		orders = []models.OrderResponse{}
	}

	// Check if order already exists, update if so
	found := false
	for i, o := range orders {
		if o.OrderID == order.OrderID {
			orders[i] = *order
			found = true
			break
		}
	}

	// If not found, append new order
	if !found {
		orders = append(orders, *order)
	}

	// Save to file
	orderFile := m.getOrderFilePath(did)
	didPath := filepath.Dir(orderFile)
	if err := os.MkdirAll(didPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal orders: %w", err)
	}

	return os.WriteFile(orderFile, data, 0644)
}

// LoadOrders loads all orders for a DID
func (m *OrderStorageManager) LoadOrders(did string) ([]models.OrderResponse, error) {
	orderFile := m.getOrderFilePath(did)

	// Check if file exists
	if _, err := os.Stat(orderFile); os.IsNotExist(err) {
		return []models.OrderResponse{}, nil
	}

	data, err := os.ReadFile(orderFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read orders file: %w", err)
	}

	var orders []models.OrderResponse
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orders: %w", err)
	}

	return orders, nil
}

// GetOrder gets a specific order by ID
func (m *OrderStorageManager) GetOrder(did, orderID string) (*models.OrderResponse, error) {
	orders, err := m.LoadOrders(did)
	if err != nil {
		return nil, err
	}

	for _, order := range orders {
		if order.OrderID == orderID {
			return &order, nil
		}
	}

	return nil, fmt.Errorf("order not found: %s", orderID)
}

// UpdateOrderStatus updates the status of an order
func (m *OrderStorageManager) UpdateOrderStatus(did, orderID string, status string, txHash string) error {
	orders, err := m.LoadOrders(did)
	if err != nil {
		return err
	}

	found := false
	for i, order := range orders {
		if order.OrderID == orderID {
			orders[i].Status = status
			if txHash != "" {
				orders[i].TxHash = txHash
			}
			orders[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("order not found: %s", orderID)
	}

	return m.SaveOrders(did, orders)
}

// SaveOrders saves the entire order list (used for batch updates)
func (m *OrderStorageManager) SaveOrders(did string, orders []models.OrderResponse) error {
	orderFile := m.getOrderFilePath(did)
	didPath := filepath.Dir(orderFile)
	
	if err := os.MkdirAll(didPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal orders: %w", err)
	}

	return os.WriteFile(orderFile, data, 0644)
}

// DeleteOrder removes an order from storage
func (m *OrderStorageManager) DeleteOrder(did, orderID string) error {
	orders, err := m.LoadOrders(did)
	if err != nil {
		return err
	}

	var updatedOrders []models.OrderResponse
	found := false
	for _, order := range orders {
		if order.OrderID != orderID {
			updatedOrders = append(updatedOrders, order)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("order not found: %s", orderID)
	}

	return m.SaveOrders(did, updatedOrders)
}
