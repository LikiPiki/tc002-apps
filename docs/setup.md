# Настройка часов

Проект ориентирован на **Ulanzi TC002 с AWTRIX NG**, а не штатную прошивку Ulanzi и не AWTRIX 3.

1. Установить AWTRIX NG по [официальной инструкции TC002](https://blueforcer.github.io/awtrix-ng/tc002/getting-started/tc002/).
2. Подключить часы к Wi-Fi и открыть их веб-интерфейс.
3. Узнать IP-адрес часов и проверить доступность с машины, запускающей Go-команды.
4. Отправить уведомление:

   ```sh
   AWTRIX_URL=http://192.168.1.50 go run ./cmd/clockctl --text 'Hello TC002'
   ```

Первоначальный клиент рассчитан на доступный без авторизации HTTP API. Если в вашей конфигурации нужна авторизация, её поддержку нужно добавить перед использованием.

Berry-приложения добавляются во вкладке Scripts. Справочник: [AWTRIX NG scripting](https://blueforcer.github.io/awtrix-ng/guides/scripting/).
