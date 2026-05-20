"""Result formatting and display functions."""

import json

from .config import _c


def _print_entry_body(r, indent=""):
    """Print definition, examples, synonyms, and extra_notes for one entry."""
    if r.get("cn_definition"):
        print(f"{indent}{_c('label', '释义:')} {r['cn_definition']}")
    for en, cn in r.get("examples", []):
        if en and cn:
            print(f"{indent}{_c('label', '例:')} {en} / {cn}")
        elif en:
            print(f"{indent}{_c('label', '例:')} {en}")
        elif cn:
            print(f"{indent}{_c('label', '例译:')} {cn}")
    synonyms = r.get("synonyms", [])
    if synonyms:
        syn_text = _c('dim', ', ').join(_c('word', s) for s in synonyms)
        print(f"{indent}{_c('label', '同义:')} {syn_text}")
    antonyms = r.get("antonyms", [])
    if antonyms:
        ant_text = _c('dim', ', ').join(_c('word', s) for s in antonyms)
        print(f"{indent}{_c('label', '反义:')} {ant_text}")
    extra = r.get("extra_notes", "")
    if extra:
        try:
            notes = json.loads(extra)
            for note in notes:
                note_type = note.get("type", "")
                type_label = {
                    "usage": "用法", "drv": "派生", "regional": "注",
                    "sense": "释义补充", "quotation": "名言",
                    "phrase": "短语", "note": "注",
                }.get(note_type, note_type)
                en = note.get("en", "")
                cn = note.get("cn", "")
                en_lines = en.split("\n")
                cn_lines = cn.split("\n")
                max_lines = max(len(en_lines), len(cn_lines))
                print(f"{indent}{_c('label', f'[{type_label}]')}")
                for i in range(max_lines):
                    en_part = en_lines[i].rstrip() if i < len(en_lines) else ""
                    cn_part = cn_lines[i].rstrip() if i < len(cn_lines) else ""
                    if en_part and cn_part:
                        print(f"{indent}{en_part} / {cn_part}")
                    elif en_part:
                        print(f"{indent}{en_part}")
                    elif cn_part:
                        print(f"{indent}{cn_part}")
                    else:
                        print()
        except (json.JSONDecodeError, TypeError):
            pass


def print_results_english(results):
    source_names = {"collins": "柯林斯", "oxford": "牛津"}

    for r in results:
        src_label = source_names.get(r["source"], r["source"])
        pos_str = f" {r['pos']}" if r["pos"] else ""
        pron_str = ""
        if r.get("pronunciation"):
            try:
                ipas = json.loads(r["pronunciation"])
                if ipas:
                    pron_str = f"{_c('dim', ' /')}{_c('pron', ' | '.join(ipas))}{_c('dim', '/')}"
            except (json.JSONDecodeError, TypeError):
                pass
        header = f"{_c('source', src_label)}: 【{_c('word', r['word'])}】{pron_str}{_c('dim', pos_str)}"
        if r["cross_ref"]:
            header += f"  -> see: {r['cross_ref']}"
        print(header)

        _print_entry_body(r)
        print()
        print()


def print_results_chinese(results):
    source_names = {"collins": "柯林斯", "oxford": "牛津"}

    for (src, word, cn_def), examples in results.items():
        src_label = source_names.get(src, src)
        print(f"{_c('source', src_label)}: 【{_c('word', word)}】 {cn_def}")
        for ex in examples[:3]:
            print(f"  {ex}")
        print()
