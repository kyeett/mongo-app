package main

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"testing"
	"time"
)

var (
	testMongoURI = "mongodb://localhost:27017"
)

// Test simple insert/get
func TestPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := NewConnectedMongoClient(testMongoURI)
	require.NoError(t, err)

	s := service{
		client: client,
	}

	data := bson.D{
		{"name", "magnus"},
		{"age", 123},
	}

	collectionName := uuid.NewString()
	err = s.InsertData(ctx, collectionName, data)
	require.NoError(t, err)

	dbData, err := s.GetData(ctx, collectionName)
	require.NoError(t, err)

	dbMap := dbData.Map()

	require.Len(t, dbMap, 3)
	require.Equal(t, dbMap["name"], "magnus")
	require.Equal(t, dbMap["age"], int32(123))
}
