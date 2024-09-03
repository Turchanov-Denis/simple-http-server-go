package taskstore

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Task struct {
	Id   int       `bson:"id"`
	Text string    `bson:"text"`
	Tags []string  `bson:"tags"`
	Due  time.Time `bson:"due"`
}

type MongoTaskStore struct {
	client     *mongo.Client
	collection *mongo.Collection
	ctx        context.Context
}

// Создание нового хранилища
func NewMongo(uri, dbName, collectionName string) (*MongoTaskStore, error) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to mongo: %w", err)
	}

	collection := client.Database(dbName).Collection(collectionName)

	return &MongoTaskStore{
		client:     client,
		collection: collection,
		ctx:        ctx,
	}, nil
}

// Создание задачи
func (s *MongoTaskStore) CreateTask(text string, tags []string, due time.Time) int {
	task := Task{
		Id:   s.nextID(),
		Text: text,
		Tags: tags,
		Due:  due,
	}

	_, err := s.collection.InsertOne(s.ctx, task)
	if err != nil {
		panic(err)
	}

	return task.Id
}

// Генерация нового ID через коллекцию-счётчик
func (s *MongoTaskStore) nextID() int {
	counter := s.client.Database(s.collection.Database().Name()).Collection("counters")
	filter := bson.M{"_id": "taskid"}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result struct {
		Seq int `bson:"seq"`
	}
	err := counter.FindOneAndUpdate(s.ctx, filter, update, opts).Decode(&result)
	if err != nil {
		panic(err)
	}
	return result.Seq
}

// Получение всех задач
func (s *MongoTaskStore) GetAllTasks() []Task {
	cursor, err := s.collection.Find(s.ctx, bson.M{})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(s.ctx)

	var tasks []Task
	cursor.All(s.ctx, &tasks)
	return tasks
}

// Получение задачи по ID
func (s *MongoTaskStore) GetTask(id int) (Task, error) {
	var task Task
	err := s.collection.FindOne(s.ctx, bson.M{"id": id}).Decode(&task)
	if err == mongo.ErrNoDocuments {
		return Task{}, fmt.Errorf("task with id=%d not found", id)
	}
	return task, err
}

// Удаление одной задачи
func (s *MongoTaskStore) DeleteTask(id int) error {
	res, err := s.collection.DeleteOne(s.ctx, bson.M{"id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("task with id=%d not found", id)
	}
	return nil
}

// Удаление всех задач
func (s *MongoTaskStore) DeleteAllTasks() error {
	_, err := s.collection.DeleteMany(s.ctx, bson.M{})
	return err
}

// Получение задач по тегу
func (s *MongoTaskStore) GetTasksByTag(tag string) []Task {
	cursor, err := s.collection.Find(s.ctx, bson.M{"tags": tag})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(s.ctx)

	var tasks []Task
	cursor.All(s.ctx, &tasks)
	return tasks
}

// Получение задач по дате
func (s *MongoTaskStore) GetTasksByDueDate(year int, month time.Month, day int) []Task {
	start := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	cursor, err := s.collection.Find(s.ctx, bson.M{"due": bson.M{"$gte": start, "$lt": end}})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(s.ctx)

	var tasks []Task
	cursor.All(s.ctx, &tasks)
	return tasks
}
