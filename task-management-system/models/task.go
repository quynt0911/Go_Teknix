package models

import "gorm.io/gorm"

type Task struct {
    gorm.Model
    Title       string `json:"title"`
    Description string `json:"description"`
    Status      string `json:"status"` // pending, completed, etc.
    DueDate     string `json:"due_date"`
    Category    string `json:"category"`
    UserID      uint   `json:"user_id"`
}
