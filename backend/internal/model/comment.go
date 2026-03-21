package model

import "time"

type Comment struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"-" bson:"user_id"`
	TaskID    string    `json:"task_id" bson:"task_id"`
	TodoID    string    `json:"todo_id,omitempty" bson:"todo_id,omitempty"`
	ActorType string    `json:"actor_type" bson:"actor_type"`
	ActorID   string    `json:"actor_id" bson:"actor_id"`
	ActorName string    `json:"actor_name" bson:"actor_name"`
	Content   string    `json:"content" bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
