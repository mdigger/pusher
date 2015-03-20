package pusher

import (
	"errors"
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
func (s *Store) AddDevice(regDevice *DeviceRegister) error {
	if regDevice == nil {
		return errors.New("no parameters")
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		// открываем коллекцию данных приложения
		bucket, err := tx.CreateBucketIfNotExists([]byte(regDevice.App))
		if err != nil {
			return err
		}
		devices := make(Devices)
		// подгужаем данные пользователя, если они существуют
		if data := bucket.Get([]byte(regDevice.User)); data != nil {
			if err := codec.NewDecoderBytes(data, &codecHandle).Decode(&devices); err != nil {
				return err
			}
		}
		// добавляем токен устройства для указанного идентификатора приложения
		if !devices.Add(regDevice.Bundle, regDevice.Token) {
			return nil // Идентификатор уже был в списке — нечего сохранять
		}

		var data []byte
		// кодируем в бинарный формат
		if err := codec.NewEncoderBytes(&data, &codecHandle).Encode(devices); err != nil { // Кодируем данные для сохранения
			return err
		}
		return bucket.Put([]byte(regDevice.User), data) // Сохраняем их в хранилище
	})
}
