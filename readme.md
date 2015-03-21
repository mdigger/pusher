## Компиляция и запуск

	$ go get -u -v github.com/mdigger/pusher/server
	$ cd $GOPATH/src/github.com/mdigger/pusher/server
	$ go build
	$ ./server

При запуске сервер читает конфигурационный файл `pusher.json`, устанавливает соединение со всеми описанными в нем APNS-конфигурациями соединения (для проверки) и стартует HTTP-сервер. Если что-то не сложится, то сервис выдаст ошибку.

В лог выводится вся информация о взаимодействии с push-серверами.

## Конфигурация

Конфигурация описывается в файле `pusher.json`. Например:

	{
		"db": "users.db",
		"server": "localhost:8080",
		"apps": {
			"push-test": {
				"com.xyzrd.PushTest": {
					"type": "apns",
					"sandbox": true,
					"certificate": [
					"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZpVENDQkhHZ0F3SUJBZ0lJR01vWi8wSlhScU13RFFZSktvWklodmNOQ
					...kUQpLM3IxUmZNUE4wajFUeXlaVjRkSXVZbE5mbHRWczBrR1hha3ozL1U9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
					"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZpRENDQkhDZ0F3SUJBZ0lJVUJCazhHczBCbnd3RFFZSktvWklodmNOQ
					...PWgp1OExUNVI2RXB2SUxlVDVvNUZTRzluNk94UGZFcTRBSGNnUERtZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
					],
					"privateKey": "LS0tLNVI2RXS1FCRUdJTiBSU0EgUFJJVkFURSBLRVktLS0t...RCBSU0EgUFJJVkFURSBLRVktLS0tLQo="
				}
			}
		}
	}

- `db` -- имя файла с базой данных (создастся автоматически, если нет). По умолчанию `pusher.db` (можно опустить).
- `server` -- адрес и порт на котором будет запущен сервер. По умолчанию `localhost:8080` (можно опустить).
- `apps` -- список поддерживаемых сервисов (имена используются как часть URL в запросе). 
- далее идет список **bundleId** приложений. В данном примере это `com.xyzrd.PushTest`.
- внутри -- описание конфигурации для подключения, включая сертификаты.

## Генерация конфигурационного описания

Чтобы не возиться с ручным созданием конфигурационного файла, я сделал небольшое приложение для автоматизации этого процесса для APNS-сервисов.

	$ cd $GOPATH/src/github.com/mdigger/apns/config/
	$ go build

Далее, необходимо скопировать туда два файла с сертификатами: `cert.pem` и `key.pem`. Если сертификаты запаролены, то пароли с них должны быть сняты. После этого можно запустить само приложение:

	$ ./config --help
	Usage of ./config:
  		-bundle="": bundle id (if empty trying to find in certificate file info)
  		-cert="cert.pem": certificate file name
  		-key="key.pem": private key file name
  		-output="config.json": output filename
  		-sandbox=true: sandbox mode
  	$./config

В этом же каталоге должен появиться файл `config.json`:

	{
		"type": "apns",
		"bundleId": "com.xyzrd.PushTest",
		"sandbox": true,
		"certificate": [
			"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZpVENDQkhHZ0F3SUJBZ0lJR01vWi8wSlhScU13RFFZSktvWklodmNOQVFFRkJRQ...yWHZkUQpLM3IxUmZNUE4wajFUeXlaVjRkSXVZbE5mbHRWczBrR1hha3ozL1U9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K",
			"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUZpRENDQkhDZ0F3SUJBZ0lJVUJCazhHczBCbnd3RFFZSktvWklodmNOQVFFRkJRQ...BRjZPWgp1OExUNVI2RXB2SUxlVDVvNUZTRzluNk94UGZFcTRBSGNnUERtZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
		],
		"privateKey": "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBdzZIMTg5R1kzUllVSENpQy9xbEJEci8...lIydEdmcEZNPQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo="
	}

Из него можно скопировать все нужные части в наш конфигурационный файл.

## Обращение к серверу

Для каждого описанного в конфигурации сервиса доступны отдальные URL:

	$ curl -X "GET" "http://localhost:8080/"

	[
	  "push-test"
	]

	$ curl -X "GET" "http://localhost:8080/push-test"

	[
	  "com.xyzrd.PushTest"
	]

Для регистрации пользователей и привязки токенов устройств можно воспользоваться следующим запросом:

	curl -X "POST" "http://localhost:8080/push-test/register" \
		-H "Content-Type: application/json" \
		-d $'{
	  			"user": "dmitrys",
	  			"bundle": "com.xyzrd.PushTest",
	  			"token": "B8108B88198789E9696E11A2FFE9710B776A9851673C2FDEDFCE1BE318AE7C90"
			}'

Строка `"Content-Type: application/json"` в заголовке является обязательной: без нее не понятен формат передаваемых данных.

Отправка push-сообщений:

	curl -X "POST" "http://localhost:8080/push-test/push" \
		-H "Content-Type: application/json" \
		-d $'{
			  "users": [
			    "dmitrys"
			  ],
			  "messages": {
			    "com.xyzrd.PushTest": {
			      "payload": {
			        "aps": {
			          "alert": "test message",
			          "badge": 5
			        }
			      }
			    }
			  }
			}'

- `users` -- список пользователей, которым нужно отправить сообщение
- `messages` -- список push-сообщений с привязкой к **bundleId**