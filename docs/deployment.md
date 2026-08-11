# Руководство по развертыванию (Deployment Guide)

Данное руководство описывает процесс установки и запуска `nextcloud-mcp-gateway` в локальной среде на основе конфигурации `pyproject.toml`.

## Требования
- Python 3.10+
- Git

## Переменные окружения (`.env`)

Для конфигурации сервиса используются переменные окружения, которые можно задать в файле `.env` в корне проекта (скопировав `.env.example`). Модуль `config.py` парсит эти значения.

| Переменная | Описание | Значение по умолчанию | Обязательно |
| :--- | :--- | :--- | :--- |
| `NC_URL` | URL локального/внутреннего инстанса Nextcloud | `http://127.0.0.1:8080` | Нет |
| `NC_PUBLIC_URL` | Публичный URL инстанса Nextcloud | `https://nc.shtab-ai.ru` | Нет |
| `NC_USER` | Имя пользователя Nextcloud | `""` | **Да** (для авторизации) |
| `NC_APP_PASSWORD`| Пароль приложения (App Password) Nextcloud | `""` | **Да** (для авторизации) |
| `NC_TIMEOUT` | Таймаут (в секундах) для HTTP-запросов | `30.0` | Нет |

---

## Пошаговая инструкция по локальному запуску

**1. Клонирование репозитория:**
```bash
git clone https://github.com/TheNovaNodes/nextcloud-mcp-gateway.git
cd nextcloud-mcp-gateway
```

**2. Настройка окружения:**
```bash
cp .env.example .env
# Отредактируйте .env, добавив ваши NC_USER и NC_APP_PASSWORD
nano .env
```

**3. Установка зависимостей (включая dev-инструменты):**
Установка производится в "editable" режиме, согласно конфигурации `pyproject.toml`.
```bash
pip install -e ".[dev]"
```

**4. Запуск сервера:**
Поскольку в `pyproject.toml` определен скрипт `nextcloud-mcp-gateway = "nextcloud_mcp_gateway.server:main"`, после установки вы можете запустить сервер командой:
```bash
nextcloud-mcp-gateway
```
*Альтернативный способ запуска:*
```bash
python3 -m nextcloud_mcp_gateway.server
```

---

## Интеграция с MCP-клиентом (Claude Desktop)

Для использования шлюза ИИ-агентом (например, Claude Desktop), необходимо обновить конфигурацию клиента (`claude_desktop_config.json`), передав нужные переменные окружения напрямую:

```json
{
  "mcpServers": {
    "nextcloud-gateway": {
      "command": "python3",
      "args": ["-m", "nextcloud_mcp_gateway.server"],
      "cwd": "/путь/к/папке/репозитория/nextcloud-mcp-gateway",
      "env": {
        "NC_URL": "http://127.0.0.1:8080",
        "NC_PUBLIC_URL": "https://nc.shtab-ai.ru",
        "NC_USER": "ваше_имя_пользователя",
        "NC_APP_PASSWORD": "ваш_пароль_приложения"
      }
    }
  }
}
```

---

### TODO: Docker & Инфраструктура
> В данный момент репозиторий не содержит `Dockerfile` или `docker-compose.yml`.
>
> **Необходимо реализовать:**
> - Создание `Dockerfile` на базе легковесного образа Python 3.10+ (например, `python:3.10-slim`).
> - Настройку Docker-сети (networks) для безопасного общения контейнера `nextcloud-mcp-gateway` с контейнерами локального Nextcloud.
> - Описание конфигурации проброса портов (ports) и томов (volumes) при необходимости, а также взаимодействие между управляющими и exit-нодами, если таковая топология будет внедрена.