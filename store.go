package pusher

import (
	"bytes"
	"encoding/gob"
	"log"

	"github.com/boltdb/bolt"
)

// Store описывает хранилище данных.
type Store struct {
	db    *bolt.DB   // хранилище
	stats bolt.Stats // статистика базы
}

// OpenStore открывает и возвращает хранилище данных.
func OpenStore(filename string) (*Store, error) {
	db, err := bolt.Open(filename, 0666, nil)
	if err != nil {
		return nil, err
	}
	var store = &Store{
		db:    db,
		stats: db.Stats(), // сохраняем статистику при открытии
	}
	store.Backup()
	return store, nil
}

// Close закрывает ранее открытое хранилище данных.
func (s *Store) Close() error {
	return s.db.Close()
}

// Backup сохраняет копию базы данных.
func (s *Store) Backup() {
	if s.db == nil {
		return
	}
	var name = s.db.Path()
	if name == "" {
		return // база закрыта
	}
	db.View(func(tx *bolt.Tx) error {
		// делаем копию файла
		if err := tx.CopyFile(name+".bak", 0666); err != nil {
			log.Printf("Error database backup: %v", err)
			return err
		}
		log.Printf("DB Size: %d", tx.Size())
		return nil
	})
}

// AddDevice добавляет в хранилище информацию об идентификаторе устройства пользователя приложения.
func (s *Store) AddDevice(app, bundle, user, token string) error {
	return s.db.Batch(func(tx *bolt.Tx) error {
		// открываем коллекцию данных приложения
		bucket, err := tx.CreateBucketIfNotExists([]byte(app))
		if err != nil {
			return err
		}
		// подгружаем данные пользователя, если они существуют
		devices := make(Devices)
		if data := bucket.Get([]byte(user)); data != nil {
			if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&devices); err != nil {
				return err
			}
		}
		// добавляем токен устройства для указанного идентификатора приложения
		if !devices.Add(bundle, token) {
			return nil // Идентификатор уже был в списке — нечего сохранять
		}

		var data bytes.Buffer
		// кодируем в бинарный формат
		if err := gob.NewEncoder(&data).Encode(devices); err != nil { // Кодируем данные для сохранения
			return err
		}
		return bucket.Put([]byte(user), data.Bytes()) // Сохраняем их в хранилище
	})
}

// GetDevices возвращает для каждого пользователя список зарегистрированных для него устройств.
func (s *Store) GetDevices(app string, users ...string) (map[string]Devices, error) {
	var result = make(map[string]Devices, len(users))
	err := s.db.View(func(tx *bolt.Tx) error {
		// открываем коллекцию данных приложения
		bucket := tx.Bucket([]byte(app))
		for _, user := range users {
			// декодируем из бинарного формата
			devices := make(Devices)
			// подгружаем данные пользователя, если они существуют
			if data := bucket.Get([]byte(user)); data != nil {
				if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&devices); err != nil {
					return err
				}
			}
			result[user] = devices
		}
		return nil
	})
	return result, err
}
