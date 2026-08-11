# Справочник API и потоки данных

Этот документ описывает все инструменты (tools), предоставляемые `nextcloud-mcp-gateway` ИИ-агентам по протоколу MCP.

Все инструменты реализованы как асинхронные функции в `server.py` и используют HTTP-запросы к Nextcloud WebDAV или OCS REST API.

## Процесс санитизации и трансформации данных

- **Источник:** Запросы и параметры приходят от ИИ-агента через MCP клиент (например, Claude Desktop).
- **Санитизация путей:** Все входящие пути к файлам и директориям проходят через функцию `normalize_path(path: str) -> str`.
  - Она удаляет начальные/конечные пробелы (`.strip()`).
  - Проверяет наличие лидирующего слэша (`/`) и добавляет его, если он отсутствует.
  - Это предотвращает проблемы склейки базового WebDAV URL (`{nc_url}/remote.php/dav/files/{user}`) и пути файла.
- **Трансформация при чтении списка файлов (PROPFIND):** Nextcloud возвращает сырой XML. Сервер парсит его (с помощью `xml.etree.ElementTree`), извлекает нужные свойства (размер, дата изменения, тип ресурса) и трансформирует в чистый JSON-массив словарей для отдачи агенту.
- **Назначение:** Очищенные и преобразованные данные используются для формирования HTTP-запросов к Nextcloud. Полученные от Nextcloud данные возвращаются агенту в формате JSON.

---

## Доступные инструменты (Tools)

### 1. `list_files`
Возвращает список файлов и директорий по указанному пути, используя WebDAV команду `PROPFIND` с `Depth: 1`. Возвращает распарсенный JSON вместо исходного XML.

**Параметры:**
| Параметр | Тип   | Описание | По умолчанию |
| :--- | :--- | :--- | :--- |
| `path` | `str` | Путь к директории в Nextcloud | `"/"` |

**Успешный ответ:**
```json
{
  "status": "success",
  "path": "/Documents",
  "count": 2,
  "items": [
    {
      "href": "/remote.php/dav/files/user/Documents/report.pdf",
      "is_directory": false,
      "size_bytes": 1048576,
      "last_modified": "Mon, 10 Jul 2023 15:00:00 GMT",
      "content_type": "application/pdf"
    }
  ]
}
```

**Ошибки:**
| Код (HTTP/Status) | Описание |
| :--- | :--- |
| `404` / `error` | Директория не найдена (Path not found) |
| `401`, `403` / `error`| Ошибка авторизации (Authentication failed) |
| `parse_error` | Ошибка разбора XML-ответа от сервера |

---

### 2. `read_file`
Читает текстовое содержимое файла напрямую из Nextcloud. Использует обычный `GET` запрос к WebDAV эндпоинту.

**Параметры:**
| Параметр | Тип   | Описание | Обязательный |
| :--- | :--- | :--- | :--- |
| `path` | `str` | Путь к читаемому файлу | Да |

**Успешный ответ:**
```json
{
  "status": "success",
  "path": "/Documents/spec.md",
  "size_bytes": 1500,
  "content": "# Specification\n..."
}
```

**Ошибки:**
| Код (HTTP/Status) | Описание |
| :--- | :--- |
| `404` / `error` | Файл не найден |
| Не-200 / `error` | Другие ошибки HTTP |

---

### 3. `write_file`
Создает новый или перезаписывает существующий файл в Nextcloud, используя WebDAV `PUT`. Контент кодируется в `UTF-8`.

**Параметры:**
| Параметр | Тип   | Описание | Обязательный |
| :--- | :--- | :--- | :--- |
| `path` | `str` | Путь к файлу для записи | Да |
| `content`| `str` | Текстовое содержимое файла | Да |

**Успешный ответ:**
```json
{
  "status": "success",
  "path": "/Documents/new.txt",
  "bytes_written": 25,
  "message": "File written successfully"
}
```

**Ошибки:**
| Код (HTTP/Status) | Описание |
| :--- | :--- |
| `401`, `403` / `error`| Нет прав для записи файла (Unauthorized to write file) |

---

### 4. `delete_file`
Удаляет файл или директорию через WebDAV команду `DELETE`.

**Параметры:**
| Параметр | Тип   | Описание | Обязательный |
| :--- | :--- | :--- | :--- |
| `path` | `str` | Путь к удаляемому ресурсу | Да |

**Успешный ответ:**
```json
{
  "status": "success",
  "path": "/Documents/old.txt",
  "message": "Resource deleted"
}
```

**Ошибки:**
| Код (HTTP/Status) | Описание |
| :--- | :--- |
| `404` / `error` | Ресурс для удаления не найден |

---

### 5. `create_folder`
Создает новую директорию через WebDAV `MKCOL`.

**Параметры:**
| Параметр | Тип   | Описание | Обязательный |
| :--- | :--- | :--- | :--- |
| `path` | `str` | Путь для создания новой директории | Да |

**Успешный ответ:**
```json
{
  "status": "success",
  "path": "/NewFolder",
  "message": "Folder created successfully"
}
```

**Ошибки:**
| Код (HTTP/Status) | Описание |
| :--- | :--- |
| `405` / `exists` | Директория уже существует (Method Not Allowed) |

---

### 6. `get_user_info`
Запрашивает информацию о пользователе, включая квоту и email, используя Nextcloud OCS REST API.

**Параметры:**
Параметры не требуются.

**Успешный ответ:**
```json
{
  "status": "success",
  "user": "zavlab",
  "display_name": "Ivan Ivanov",
  "email": "ivan@example.com",
  "quota": {
    "free": 10737418240,
    "used": 1048576,
    "total": 10738466816,
    "relative": 0.01
  },
  "storage_location": "/var/www/nextcloud/data/zavlab"
}
```

---

### 7. `nextcloud_health`
Проверяет доступность эндпоинта `/status.php` и базовые параметры.

**Параметры:**
Параметры не требуются.

**Успешный ответ:**
```json
{
  "status": "healthy",
  "endpoint": "http://127.0.0.1:8080",
  "public_url": "https://nc.shtab-ai.ru",
  "authenticated": true,
  "user": "zavlab",
  "details": {
    "installed": true,
    "maintenance": false,
    "needsDbUpgrade": false,
    "version": "30.0.0.14"
  }
}
```

**Ошибки:**
| Статус ответа JSON | Описание |
| :--- | :--- |
| `maintenance` | `installed: false` - Nextcloud находится в режиме обслуживания |
| `degraded` | HTTP статус отличный от 200 (ошибка на сервере) |
| `unreachable` | Сетевая ошибка при попытке подключения |
