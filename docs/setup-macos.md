# Подготовка окружения на macOS

> Однократная настройка машины под разработку LLMProxy + on-prem RAG.

---

## Что ставим и зачем

| Инструмент       | Зачем                                                                                  |
|------------------|----------------------------------------------------------------------------------------|
| **Homebrew**     | Менеджер пакетов macOS — через него ставится всё остальное                             |
| **Docker Desktop** | Docker Engine + GUI для управления контейнерами и compose-стеком                     |
| **Go 1.21+**     | Сборка LLMGuard и Dashboard (как контейнеров — необязательно, но удобно для локальной разработки) |
| **pyenv**        | Параллельные версии Python (для on-prem RAG нужен 3.11+)                               |
| **Python 3.11**  | rag-api (FastAPI), ingest pipeline, embeddings                                         |
| **curl / jq / httpie** | Удобные CLI для тестирования API                                                 |
| **Xcode CLT**    | Компиляторы (нужны для сборки некоторых Python-зависимостей и Go cgo)                  |

---

## Шаги

### 1. Homebrew

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# На Apple Silicon после установки добавь brew в PATH (инсталлятор выведет точную команду):
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
eval "$(/opt/homebrew/bin/brew shellenv)"
```

### 2. Xcode Command Line Tools (если ещё не стоят)

```bash
xcode-select -p
# Если выводит путь /Library/Developer/CommandLineTools — уже стоит.
# Иначе:
xcode-select --install

# Проверка:
make --version
clang --version
```

### 3. Docker Desktop

```bash
brew install --cask docker
# После установки запустить приложение Docker из /Applications вручную
# и дождаться статуса "running" в menu bar.

docker version
docker compose version
```

### 4. Go

```bash
brew install go
go version   # Должно быть >= 1.21 (актуально из brew — обычно 1.23+)
```

### 5. pyenv + Python 3.11

```bash
brew install pyenv

# Подключить pyenv в zsh
cat >> ~/.zshrc <<'EOF'

# pyenv
export PYENV_ROOT="$HOME/.pyenv"
[[ -d $PYENV_ROOT/bin ]] && export PATH="$PYENV_ROOT/bin:$PATH"
eval "$(pyenv init -)"
EOF

source ~/.zshrc

# Установить Python 3.11.x (последний патч)
pyenv install 3.11
pyenv global 3.11

python3 --version    # → Python 3.11.x
```

Если `pyenv install 3.11` падает с ошибками сборки зависимостей:

```bash
brew install openssl readline sqlite3 xz zlib
pyenv install 3.11
```

### 6. pip и venv

```bash
python3 -m ensurepip --upgrade
python3 -m pip install --upgrade pip

# venv — стандартный модуль, отдельно ставить не нужно. Создание окружения:
# python3 -m venv .venv
```

### 7. Утилиты для API-тестов

```bash
brew install curl jq httpie

curl --version
jq --version
http --version
```

---

## Python-зависимости для on-prem RAG

В корне проекта уже есть готовый скрипт `setup_rag_venv.sh`:

```bash
./setup_rag_venv.sh
```

Что он делает:

```bash
#!/usr/bin/env bash
set -euo pipefail
VENV_DIR="${VENV_DIR:-.venv}"

[[ -d "$VENV_DIR" ]] || python3 -m venv "$VENV_DIR"
source "${VENV_DIR}/bin/activate"

python -m pip install --upgrade pip
pip install \
  sentence-transformers \
  tiktoken \
  fastapi \
  'uvicorn[standard]' \
  langfuse
```

В новых терминалах активация:

```bash
source .venv/bin/activate
```

---

## Проверка готовности

```bash
brew --version
docker version
docker compose version
go version
python3 --version
pip --version
curl --version
jq --version
http --version
xcode-select -p
```

Если все команды отвечают без ошибок — окружение готово. Возвращайся к [README](../README.md) → Быстрый старт.
