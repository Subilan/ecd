"""Shared utilities: tabfile extraction, HTML helpers, IPA cleaning."""

import re
import subprocess
import sys

from .config import PYGLOSSARY


def extract_tabfile(dict_path, output_path):
    result = subprocess.run(
        [PYGLOSSARY, dict_path, output_path, "--read-format", "AppleDictBin"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        result.check_returncode()


_PUNCT_CLEAN_RE = re.compile(r"\s+([,.;:?!])|\(\s+|\s+\)")


def _clean_spacing(text):
    """Remove spurious spaces around English punctuation introduced by
    joining separate HTML text nodes with spaces."""
    return _PUNCT_CLEAN_RE.sub(lambda m: m.group(0).strip(), text)


def itertext(el):
    parts = []
    for t in el.itertext():
        s = t.strip()
        if s:
            parts.append(s)
    return _clean_spacing(" ".join(parts))


def child_elements(el, tag=None, class_contains=None):
    """Yield direct children of el matching optional tag and class substring."""
    for child in el:
        if tag and child.tag != tag:
            continue
        if class_contains:
            cls = child.get("class", "") or ""
            if class_contains not in cls.split():
                continue
        yield child


def clean_ipa(text):
    """Strip HTML markup from IPA pronunciation text."""
    if not text:
        return ""
    text = re.sub(r"\s+", " ", text).strip()
    text = re.sub(r"<[^>]+>", "", text)
    return text.strip()
