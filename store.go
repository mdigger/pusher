package pusher

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/cznic/ql/driver"
)

// Store описывает хранилище данных.
type Store struct {
	db *sql.DB // хранилище
}

// OpenStore открывает и возвращает хранилище данных.
func OpenStore(filename string) (*Store, error) {
	db, err := sql.Open("ql", filename) // открываем файл с базой данных
	if err != nil {
		return nil, err
	}
	tx, err := db.Begin() // открываем новую транзакцию
	if err != nil {
		return nil, err
	}
	// создаем таблицы и индексы, если они еще не были созданы
	if _, err := tx.Exec(
		`CREATE TABLE IF NOT EXISTS devices (
	app string NOT NULL,
	bundle string NOT NULL,
	user string NOT NULL,
	token string NOT NULL,
);
CREATE UNIQUE INDEX IF NOT EXISTS UniqueToken ON devices (app, bundle, token);
CREATE UNIQUE INDEX IF NOT EXISTS UniqueUserToken ON devices (app, user, bundle, token);
`); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil { // завершаем транзакцию
		return nil, err
	}
	var store = &Store{
		db: db,
	}
	return store, nil
}

// Close закрывает ранее открытое хранилище данных.
func (s *Store) Close() error {
	return s.db.Close()
}

// Backup сохраняет копию базы данных.
func (s *Store) Backup() {
	// if s.db == nil {
	// 	return
	// }
	// var name = s.db.Path()
	// if name == "" {
	// 	return // база закрыта
	// }
	// s.db.View(func(tx *bolt.Tx) error {
	// 	// делаем копию файла
	// 	if err := tx.CopyFile(name+".bak", 0666); err != nil {
	// 		log.Printf("Error database backup: %v", err)
	// 		return err
	// 	}
	// 	log.Printf("DB Size: %d", tx.Size())
	// 	return nil
	// })
}

// AddDevice добавляет в хранилище информацию об идентификаторе устройства пользователя приложения.
func (s *Store) AddDevice(app, bundle, user, token string) error {
	log.Printf("AddDevice: [%s] %s %s %s", app, bundle, user, token)
	tx, err := s.db.Begin() // открываем новую транзакцию
	if err != nil {
		return err
	}
	// удаляем предыдущую запись с таким токеным и добавляем новую
	if _, err := tx.Exec(
		`DELETE FROM devices WHERE app == $1 AND bundle == $2 AND token == $4;
		INSERT INTO devices (app, bundle, user, token) VALUES ($1, $2, $3, $4);`,
		app, bundle, user, token); err != nil {
		return err
	}
	return tx.Commit() // завершаем транзакцию
}

// GetDevices возвращает для каждого пользователя список зарегистрированных для него устройств.
func (s *Store) GetDevices(app string, users ...string) (map[string][]string, error) {
	log.Printf("GetDevices: [%s] %s", app, strings.Join(users, ", "))
	var result = make(map[string][]string, len(users))
	for _, user := range users {
		rows, err := s.db.Query(`SELECT bundle, token FROM devices WHERE app == $1 AND user == $2`, app, user)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			var bundle, token string
			if err := rows.Scan(&bundle, &token); err != nil {
				return result, err
			}
			log.Printf("> %s: %s = %s\n", user, bundle, token)
			if tokens, ok := result[bundle]; ok {
				tokens = append(tokens, token)
			} else {
				result[bundle] = []string{token}
			}
		}
		if err := rows.Err(); err != nil {
			return result, err
		}
	}
	return result, nil
}
