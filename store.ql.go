// // +build ql

package main

import (
	"database/sql"
	"sort"
	"time"

	_ "github.com/cznic/ql/driver"
)

// Store описывает хранилище данных с токенами пользователей.
type Store struct {
	db *sql.DB // хранилище
}

// OpenStore открывает и возвращает хранилище данных.
func OpenStore(dsn string) (*Store, error) {
	db, err := sql.Open("ql", dsn) // открываем файл с базой данных
	if err != nil {
		return nil, err
	}
	// устанавливаем соединение с базой
	if err := db.Ping(); err != nil {
		return nil, err
	}
	store := &Store{db: db}
	// инициализируем таблицы и индексы, если они еще не созданы
	if err := store.create(); err != nil {
		return nil, err
	}
	return store, nil
}

// Close закрывает ранее открытое хранилище данных.
func (s *Store) Close() error {
	return s.db.Close()
}

// Возвращает список зарегистрированных пользователей.
func (s *Store) GetUsers(id string) ([]string, error) {
	var list = make([]string, 0)
	rows, err := s.db.Query(
		"SELECT DISTINCT user FROM tokens WHERE bundle=$1;", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var user string
		if err = rows.Scan(&user); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, user)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Strings(list)
	return list, nil
}

// Удаляет информацию об указанном пользователе из хранилища.
// В ответ возвращается false, если пользователь не был зарегистрирован.
func (s *Store) DeleteUser(id, user string) (bool, error) {
	tx, err := s.db.Begin() // открываем новую транзакцию
	if err != nil {
		return false, err
	}
	result, err := tx.Exec(
		"DELETE FROM tokens WHERE bundle=$1 AND user=$2;", id, user)
	count, _ := result.RowsAffected()
	return (count > 0), tx.Commit()
}

// Возвращает список токенов пользователя.
func (s *Store) GetUserTokens(id, user string) ([]string, error) {
	var list = make([]string, 0)
	rows, err := s.db.Query(
		"SELECT DISTINCT token FROM tokens WHERE bundle=$1 AND user=$2;",
		id, user)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var token string
		if err = rows.Scan(&token); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, token)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Strings(list)
	return list, nil
}

// AddUserToken регистрирует токен с привязкой к пользователю.
// В ответ возвращает true, если это новый токен, и false, если он уже
// был зарегистрирован.
func (s *Store) AddUserToken(id, user, token string) (bool, error) {
	tx, err := s.db.Begin() // открываем новую транзакцию
	if err != nil {
		return false, err
	}
	var ok = true
	// добавляем запись о новом токене в базу
	_, err = tx.Exec(
		`INSERT INTO tokens (bundle, user, token) VALUES ($1, $2, $3);`,
		id, user, token)
	if err != nil {
		ok = false
		// в случае ошибки обновляем время у уже добавленного в базу токена
		_, err = tx.Exec(
			`UPDATE tokens SET timestamp=now() WHERE bundle=$1 AND user=$2 AND token=$3;`,
			id, user, token)
		if err != nil {
			return false, err
		}
	}
	return ok, tx.Commit()
}

// Удаляет зарегистрированный токен пользователя из хранилища.
// Если timestamp не установлен, то на время обращать внимание не стоит.
// В противном случае удаление должно производиться только тогда, когда
// токен был внесен ранее указанного времени.
func (s *Store) DeleteUserToken(id, user, token string, timestamp time.Time) error {
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	tx, err := s.db.Begin() // открываем новую транзакцию
	if err != nil {
		return err
	}
	// удаляем предыдущую запись с таким токеным и добавляем новую
	_, err = tx.Exec(
		`DELETE FROM tokens WHERE bundle=$1 AND user=$2 AND token=$3 AND timestamp<$4;`,
		id, user, token, timestamp)
	if err != nil {
		return err
	}
	return tx.Commit() // завершаем транзакцию
}

// create создает таблицу, если она не была создана до этого.
func (s *Store) create() error {
	tx, err := s.db.Begin() // открываем новую транзакцию
	if err != nil {
		return err
	}
	// создаем таблицы и индексы, если они еще не были созданы
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS tokens (
	bundle string NOT NULL,
	user string NOT NULL,
	token string NOT NULL,
	timestamp time NOT NULL DEFAULT now(),
);
CREATE UNIQUE INDEX IF NOT EXISTS UniqueToken ON tokens (bundle, token);
CREATE UNIQUE INDEX IF NOT EXISTS UniqueUserToken ON tokens (bundle, user, token);
`)
	if err != nil {
		return err
	}
	return tx.Commit() // завершаем транзакцию
}
