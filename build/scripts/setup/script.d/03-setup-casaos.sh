#!/bin/bash

set -e

BUILD_PATH=$(dirname "${BASH_SOURCE[0]}")/../../..

APP_NAME_SHORT=casaos

__get_setup_script_directory_by_os_release() {
	pushd "$(dirname "${BASH_SOURCE[0]}")/../service.d/${APP_NAME_SHORT}" >/dev/null

	{
		# shellcheck source=/dev/null
		{
			source /etc/os-release

			# If VERSION_CODENAME is missing (e.g., Ubuntu 26+ "resolute"), resolve it from ID
			if [[ -z "${VERSION_CODENAME}" ]]; then
				# Try to get codename from lsb_release, or fall back to ID
				VERSION_CODENAME=$(lsb_release -cs 2>/dev/null || true)
			fi

			{
				[[ -n "${VERSION_CODENAME}" ]] && pushd "${ID}"/"${VERSION_CODENAME}" >/dev/null
			} || {
				pushd "${ID}" >/dev/null
			} || {
                [[ -n ${ID_LIKE} ]] && for ID in ${ID_LIKE}; do
				    pushd "${ID}" >/dev/null && break
                done
			} || {
				echo "Unsupported OS: ${ID} ${VERSION_CODENAME} (${ID_LIKE})"
				exit 1
			}

			pwd

			popd >/dev/null

		} || {
			echo "Unsupported OS: unknown"
			exit 1
		}

	}

	popd >/dev/null
}

SETUP_SCRIPT_DIRECTORY=$(__get_setup_script_directory_by_os_release)
SETUP_SCRIPT_FILENAME="setup-${APP_NAME_SHORT}.sh"

SETUP_SCRIPT_FILEPATH="${SETUP_SCRIPT_DIRECTORY}/${SETUP_SCRIPT_FILENAME}"

{
    echo "🟩 Running ${SETUP_SCRIPT_FILENAME}..."
    $BASH "${SETUP_SCRIPT_FILEPATH}" "${BUILD_PATH}"
} || {
    echo "🟥 ${SETUP_SCRIPT_FILENAME} failed."
    exit 1
}

echo "✅ ${SETUP_SCRIPT_FILENAME} finished."
