# nextcloud-mcp-gateway ☁️

[![Python 3.10+](https://img.shields.io/badge/python-3.10%2B-blue)](https://www.python.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![MCP Server](https://img.shields.io/badge/MCP--Server-available-green)](https://modelcontextprotocol.io/)
[![Nextcloud](https://img.shields.io/badge/Nextcloud-30-0082c9.svg)](https://nextcloud.com/)
[![Status: Active](https://img.shields.io/badge/Status-Active-brightgreen)]

Высокопроизводительный **Model Context Protocol (MCP)** Data Plane сервер для интеграции с Nextcloud (`TheNovaNodes/nextcloud-mcp-gateway`). Позволяет ИИ-агентам читать, записывать, организовывать и управлять файлами пользователя, документацией, отчетами и активами CRM через WebDAV и Nextcloud OCS REST API.

---

## 📚 Документация (Documentation)

Исчерпывающая техническая документация для разработчиков находится в директории [`docs/`](docs/).

- 🏗 **[Архитектура и логика работы (Architecture)](docs/architecture.md)** — описание потоков данных, внутреннее устройство `server.py` и `config.py`, визуализация архитектуры.
- 🔄 **[Справочник API и Data Flow (API Reference)](docs/api-reference.md)** — описание доступных MCP-инструментов, параметров, санитизации путей и обработки ошибок.
- 🚀 **[Развертывание и конфигурация (Deployment Guide)](docs/deployment.md)** — пошаговая инструкция по локальному запуску и конфигурации переменных окружения.

---

## 🛠 Быстрый старт (Quick Start)

**Установка:**
```bash
git clone https://github.com/TheNovaNodes/nextcloud-mcp-gateway.git
cd nextcloud-mcp-gateway
pip install -e ".[dev]"
```

**Тестирование:**
```bash
python3 -m pytest -v
```

Подробную инструкцию по настройке конфигурации (включая Claude Desktop) и переменным окружения см. в [Руководстве по развертыванию](docs/deployment.md).

---

## Лицензия (License)

MIT — См. файл [LICENSE](LICENSE).
