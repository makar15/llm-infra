#!/usr/bin/env bash
# Создаёт .venv (если нет), активирует и ставит зависимости для RAG / embeddings / ingestion.
# Использование: ./setup_rag_venv.sh
# Другой каталог venv: VENV_DIR=/path/to/.venv ./setup_rag_venv.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${VENV_DIR:-${SCRIPT_DIR}/.venv}"

if [[ ! -d "$VENV_DIR" ]]; then
  python3 -m venv "$VENV_DIR"
fi

# shellcheck source=/dev/null
source "${VENV_DIR}/bin/activate"

python -m pip install --upgrade pip

pip install \
  sentence-transformers \
  tiktoken \
  fastapi \
  'uvicorn[standard]' \
  langfuse

echo "Готово. Активация в новых сессиях: source ${VENV_DIR}/bin/activate"
