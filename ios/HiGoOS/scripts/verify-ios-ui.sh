#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEVICE_NAME="${DEVICE_NAME:-iPhone 17}"
SCREENSHOT_PATH="${SCREENSHOT_PATH:-/tmp/higoos-ios-ui.png}"
BOOT_AND_LAUNCH="${BOOT_AND_LAUNCH:-1}"
RUN_TESTS="${RUN_TESTS:-1}"
DERIVED_DATA_PATH="${DERIVED_DATA_PATH:-$ROOT_DIR/build/DerivedData}"
ICON_PATH="$ROOT_DIR/HiGoOS/Assets.xcassets/AppIcon.appiconset/AppIcon-1024.png"

cd "$ROOT_DIR"

if ! command -v xcodegen >/dev/null 2>&1; then
  echo "xcodegen is required. Install it with: brew install xcodegen" >&2
  exit 1
fi

xcodegen generate
python3 - "$ICON_PATH" <<'PY'
import sys
from pathlib import Path
from PIL import Image

icon = Path(sys.argv[1])
if not icon.exists():
    raise SystemExit(f"Missing app icon: {icon}")
with Image.open(icon) as img:
    if img.size != (1024, 1024):
        raise SystemExit(f"App icon must be 1024x1024, got {img.size}")
    if img.format != "PNG":
        raise SystemExit(f"App icon must be PNG, got {img.format}")
PY

xcodebuild -project HiGoOS.xcodeproj -scheme HiGoOS -destination "generic/platform=iOS Simulator" -derivedDataPath "$DERIVED_DATA_PATH" build

if [[ "$BOOT_AND_LAUNCH" != "1" ]]; then
  exit 0
fi

DEVICE_ID="$(xcrun simctl list devices available -j | python3 -c '
import json
import sys

device_name = sys.argv[1]
payload = json.load(sys.stdin)
for devices in payload.get("devices", {}).values():
    for device in devices:
        if device.get("isAvailable") and device.get("name") == device_name:
            print(device.get("udid", ""))
            raise SystemExit(0)
' "$DEVICE_NAME")"

if [[ -z "$DEVICE_ID" ]]; then
  echo "No available simulator named '$DEVICE_NAME'." >&2
  exit 1
fi

xcrun simctl boot "$DEVICE_ID" 2>/dev/null || true
xcrun simctl bootstatus "$DEVICE_ID" -b

if [[ "$RUN_TESTS" == "1" ]]; then
  xcodebuild -project HiGoOS.xcodeproj -scheme HiGoOS -destination "platform=iOS Simulator,id=$DEVICE_ID" -derivedDataPath "$DERIVED_DATA_PATH" test
fi

APP_PATH="$DERIVED_DATA_PATH/Build/Products/Debug-iphonesimulator/HiGoOS.app"
if [[ ! -d "$APP_PATH" ]]; then
  echo "Built HiGoOS.app was not found in DerivedData." >&2
  exit 1
fi

xcrun simctl install "$DEVICE_ID" "$APP_PATH"
xcrun simctl launch "$DEVICE_ID" com.hiveton.higoos >/dev/null
sleep 2
xcrun simctl io "$DEVICE_ID" screenshot "$SCREENSHOT_PATH" >/dev/null
echo "Screenshot: $SCREENSHOT_PATH"
