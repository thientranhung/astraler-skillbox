# Large Plugin Settings Fixture

Generate the oversized settings file inside the run folder so the repository
does not carry a large binary/text blob:

```sh
python3 - <<'PY'
from pathlib import Path
p = Path("settings.json")
p.write_text("{" + "\"x\":\"" + ("a" * (1024 * 1024 + 1)) + "\"}")
PY
```
