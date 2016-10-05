package main

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
)

var boltOptions = &bolt.Options{
	Timeout: time.Second * 5,
}

// Store описывает хранилище токенов устройств.
type Store struct {
	db *bolt.DB
}

// OpenStore открывает хранилище токенов устройств.
func OpenStore(dsn string) (*Store, error) {
	db, err := bolt.Open(dsn, 0600, boltOptions)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close закрывает хранилище токенов устройств.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// bucketName добавляет к имени темы ~ в начале, если это sandbox.
func bucketName(topic string, sandbox bool) []byte {
	if sandbox {
		return []byte("~" + topic)
	}
	return []byte(topic)
}

// userValue возвращает текущее время в виде количества секунд и имя
// пользователя в бинарном виде.
func userValue(user string) []byte {
	var data = make([]byte, len(user)+8)
	n := binary.PutVarint(data, time.Now().Unix())
	n += copy(data[n:], []byte(user))
	return data[:n]
}

// timeAndName распаковывает из бинарного вида время и имя пользователя.
func timeAndName(data []byte) (time.Time, string) {
	sec, n := binary.Varint(data)
	return time.Unix(sec, 0), string(data[n:])
}

// Save сохраняет токен устройства в хранилище.
func (s *Store) Save(user, topic, token string, sandbox bool) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(bucketName(topic, sandbox))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(token), userValue(user))
	})
}

// GetUserTopicTokens возвращает список токенов пользователя для указанной темы.
func (s *Store) GetUserTopicTokens(topic string, sandbox bool, users ...string) ([]string, error) {
	if len(users) == 0 {
		return nil, nil
	}
	usersList := make(map[string]struct{}, len(users))
	for _, user := range users {
		usersList[user] = struct{}{}
	}
	var list = make([]string, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName(topic, sandbox))
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			_, u := timeAndName(v)
			if _, ok := usersList[u]; ok {
				list = append(list, string(k))
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

// Remove удаляет токен из хранилища, если он был добавлен после указанной даты.
func (s *Store) Remove(token, topic string, timestamp time.Time, sandbox bool) error {
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(bucketName(topic, sandbox))
		if bucket == nil {
			return nil
		}
		key := []byte(token)
		added, _ := timeAndName(bucket.Get(key))
		if added.Before(timestamp) {
			bucket.Delete(key)
		}
		return nil
	})
}

// MarshalJSON возвращает путь к хранилищу в виде строки JSON.
func (s *Store) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.db.Path())
}

// UnmarshalJSON получает путь к хранилищу из строки JSON и открывает его как
// хранилище токенов устройств.
func (s *Store) UnmarshalJSON(data []byte) error {
	var path string
	err := json.Unmarshal(data, &path)
	if err != nil {
		return err
	}
	if s.db != nil {
		s.db.Close()
	}
	store, err := bolt.Open(path, 0600, boltOptions)
	if err != nil {
		return err
	}
	s.db = store
	return nil
}
