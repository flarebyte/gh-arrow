"""Reject pending or mutable reserve revisions before running Arrow."""

import os
import re
import sys


if re.fullmatch(r"[0-9a-f]{40}", os.environ.get("RESERVE_REF", "")) is None:
    sys.exit(
        "RESERVE_REF must be a full 40-character commit SHA containing the shared "
        "configs and available on GitHub. The reserve pin is pending; set it after "
        "review, commit, and publication."
    )
