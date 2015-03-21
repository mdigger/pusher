package pusher

import (
	"github.com/boltdb/bolt"
	"github.com/ugorji/go/codec"
)

var codecHandle codec.CborHandle // параметры кодирования данных

// Store описывает хранилище данных.
type Store struct {
	db *bolt.DB // хранилище
}

// OpenStore открывает и возвращает хранилище данных.
func OpenStore(filename string) (*Store, error) {
	db, err := bolt.Open(filename, 0666, nil)
	if err != nil {
		return nil, err
	}
	return &Store{
		db: db,
	}, nil
}

// Close закрывает ранее открытое хранилище данных.
func (s *Store) Close() error {
	return s.db.Close()
}

// AddDevice добавляет в хранилище информацию об идентификаторе устройства пользователя приложения.
func (s *Store) AddDevice(app, bundle, user, token string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		// открываем коллекцию данных приложения
		bucket, err := tx.CreateBucketIfNotExists([]byte(app))
		if err != nil {
			return err
		}
		// подгужаем данные пользователя, если они существуют
		devices := make(Devices)
		if data := bucket.Get([]byte(user)); data != nil {
			if err := codec.NewDecoderBytes(data, &codecHandle).Decode(&devices); err != nil {
				return err
			}
		}
		// добавляем токен устройства для указанного идентификатора приложения
		if !devices.Add(bundle, token) {
			return nil // Идентификатор уже был в списке — нечего сохранять
		}

		var data []byte
		// кодируем в бинарный формат
		if err := codec.NewEncoderBytes(&data, &codecHandle).Encode(devices); err != nil { // Кодируем данные для сохранения
			return err
		}
		return bucket.Put([]byte(user), data) // Сохраняем их в хранилище
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
				if err := codec.NewDecoderBytes(data, &codecHandle).Decode(&devices); err != nil {
					return err
				}
			}
			result[user] = devices
		}
		return nil
	})
	return result, err
}
