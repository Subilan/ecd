#!/usr/bin/env python3
"""Build ecd.db from Apple Dictionary bundles.

Usage: extract/.venv/bin/python build_db.py [--only collins|oxford]
Setup:  python3 -m venv extract/.venv && extract/.venv/bin/pip install -r extract/requirements.txt
"""

import argparse
import copy
import json
import os
import re
import shutil
import sqlite3
import subprocess
import sys
import tempfile
from pathlib import Path
from lxml import html

EXTRACT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = EXTRACT_DIR.parent

DEFAULT_COLLINS = os.path.expanduser(
    "~/Library/Dictionaries/柯林斯高阶英汉双解词典.dictionary"
)
DEFAULT_OXFORD = os.path.expanduser(
    "~/Library/Dictionaries/牛津高阶英汉双解词典（第8版）.dictionary"
)
DEFAULT_DB = str(PROJECT_ROOT / "ecd.db")
SCHEMA_SQL = str(EXTRACT_DIR / "schema.sql")

# Prefer pyglossary from same venv as this python; fall back to PATH
PYGLOSSARY = shutil.which(
    "pyglossary", path=str(Path(sys.executable).parent) + os.pathsep + os.environ.get("PATH", "")
)
if not PYGLOSSARY:
    sys.exit("pyglossary not found. Run: extract/.venv/bin/pip install -r extract/requirements.txt")


def extract_tabfile(dict_path, output_path):
    result = subprocess.run(
        [PYGLOSSARY, dict_path, output_path, "--read-format", "AppleDictBin"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        result.check_returncode()


def itertext(el):
    parts = []
    for t in el.itertext():
        s = t.strip()
        if s:
            parts.append(s)
    return " ".join(parts)


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


def _clean_ipa(text):
    """Strip HTML markup from IPA pronunciation text."""
    if not text:
        return ""
    text = re.sub(r"\s+", " ", text).strip()
    text = re.sub(r"<[^>]+>", "", text)
    return text.strip()


def _extract_collins_pronunciation(tree):
    """Extract pronunciation from Collins word_entry. Returns JSON string or None."""
    pron_el = tree.cssselect(".word_entry > .pron")
    if not pron_el:
        return None
    ipas = []
    for type_el in pron_el[0].cssselect(".type_uk, .type_us"):
        text = _clean_ipa(type_el.text_content())
        if text:
            ipas.append(text)
    return json.dumps(ipas, ensure_ascii=False) if ipas else None


def _extract_oxford_pronunciation(container_el, top_g=None):
    """Extract pronunciation from an Oxford element (p-g or top-g).

    Checks container_el for .ei-g first; if not found and top_g is given,
    falls back to top_g > .ei-g. Returns JSON string or None.
    """
    ei_g = container_el.cssselect(".ei-g")
    if not ei_g and top_g is not None:
        ei_g = top_g.cssselect(".ei-g")
    if not ei_g:
        return None
    ipas = []
    for phon in ei_g[0].cssselect(".phon-gb, .phon-usgb, .phon-us"):
        text = _clean_ipa(phon.text_content())
        if text:
            ipas.append(text)
    return json.dumps(ipas, ensure_ascii=False) if ipas else None


def parse_collins_xref(tree, word):
    captions = tree.cssselect(".caption")
    caption_text = captions[0].text_content().strip() if captions else ""

    cross_ref = ""
    if captions:
        b_tags = captions[0].cssselect("b")
        if len(b_tags) >= 2:
            cross_ref = b_tags[-1].text_content().strip()
        else:
            m = re.search(
                r"of\s+['\"]?(\w+(?:\s+\w+)?)['\"]?[\s.]*$", caption_text
            )
            if m:
                cross_ref = m.group(1).strip()

    return [
        {
            "word": word,
            "pos": "",
            "cn_definition": caption_text,
            "cross_ref": cross_ref,
            "sense_order": 1,
            "examples": [],
            "pronunciation": None,
            "extra_notes": None,
        }
    ]


def _extract_notes_from_figures(figures):
    """Extract extra_notes from figure.note elements. Returns list of note dicts."""
    notes = []
    for fig in figures:
        cls = fig.get("class", "")
        note_type = ""
        for part in cls.split():
            if part.startswith("type-"):
                note_type = part.replace("type-", "")
                break
        # Format 1: <li><p>en</p><p>cn</p></li>
        had_li = False
        for li in fig.cssselect("li"):
            had_li = True
            ps = li.cssselect("p")
            if len(ps) >= 2:
                en_text = itertext(ps[0])
                cn_text = itertext(ps[1])
                if en_text or cn_text:
                    notes.append(
                        {"type": note_type, "en": en_text, "cn": cn_text}
                    )
        # Format 2: inline text with .def_cn spans (e.g. regional notes)
        if not had_li:
            fig_copy = copy.deepcopy(fig)
            def_cn_spans = fig_copy.cssselect(".def_cn")
            cn_parts = []
            for s in def_cn_spans:
                cn_parts.append(s.text_content().strip())
                parent = s.getparent()
                if parent is not None:
                    parent.remove(s)
            cn_text = " ".join(p for p in cn_parts if p)
            en_text = itertext(fig_copy)
            if en_text or cn_text:
                notes.append(
                    {"type": note_type, "en": en_text, "cn": cn_text}
                )
    return notes


def parse_collins_regular(tree, word):
    entries = []
    en_cn_divs = tree.cssselect(".collins_en_cn")

    for div in en_cn_divs:
        st_el = div.cssselect(".st")
        pos = st_el[0].text_content().strip() if st_el else ""

        def_cn_el = div.cssselect(".def_cn")
        cn_def = def_cn_el[0].text_content().strip() if def_cn_el else ""

        examples = []
        for li in div.cssselect("li"):
            if list(li.iterancestors("figure")):
                continue
            ps = li.cssselect("p")
            if len(ps) >= 2:
                en_text = itertext(ps[0])
                cn_text = itertext(ps[1])
                if en_text or cn_text:
                    examples.append((en_text, cn_text))

        # Extract notes from figure.note inside this collins_en_cn
        en_cn_notes = _extract_notes_from_figures(div.cssselect("figure.note"))
        en_cn_notes_json = (
            json.dumps(en_cn_notes, ensure_ascii=False) if en_cn_notes else None
        )

        if not cn_def and not examples and not en_cn_notes:
            continue

        same_count = sum(1 for e in entries if e["pos"] == pos)
        entries.append(
            {
                "word": word,
                "pos": pos,
                "cn_definition": cn_def,
                "cross_ref": None,
                "sense_order": same_count + 1,
                "examples": examples,
                "pronunciation": None,
                "extra_notes": en_cn_notes_json,
            }
        )

    # Collect figure.note elements NOT inside any .collins_en_cn
    # (e.g. quotation notes that are siblings of .collins_en_cn)
    orphan_figs = []
    for fig in tree.cssselect("figure.note"):
        inside_en_cn = any(
            "collins_en_cn" in (anc.get("class", "") or "").split()
            for anc in fig.iterancestors()
        )
        if not inside_en_cn:
            orphan_figs.append(fig)

    if orphan_figs and entries:
        orphan_notes = _extract_notes_from_figures(orphan_figs)
        if orphan_notes:
            orphan_json = json.dumps(orphan_notes, ensure_ascii=False)
            # Add to first entry with a definition; if none have one, to first entry
            target = entries[0]
            for e in entries:
                if e["cn_definition"]:
                    target = e
                    break
            existing = target.get("extra_notes")
            if existing:
                existing_list = json.loads(existing)
                existing_list.extend(orphan_notes)
                target["extra_notes"] = json.dumps(
                    existing_list, ensure_ascii=False
                )
            else:
                target["extra_notes"] = orphan_json

    return entries


def parse_oxford_xref(tree, word):
    xr_g = tree.cssselect(".sense-g .xr-g")

    if xr_g:
        cn_def = re.sub(r"\s+", " ", xr_g[0].text_content().strip())
        cross_ref = ""
        xr_links = xr_g[0].cssselect(".xr a, .Ref a")
        if xr_links:
            cross_ref = xr_links[0].text_content().strip()
        else:
            a_links = xr_g[0].cssselect("a")
            if a_links:
                cross_ref = a_links[0].text_content().strip()
        return [
            {
                "word": word,
                "pos": "",
                "cn_definition": cn_def,
                "cross_ref": cross_ref,
                "sense_order": 1,
                "examples": [],
                "pronunciation": None,
                "extra_notes": None,
            }
        ]

    # Derived-form xref: .derived with a link to parent word
    derived = tree.cssselect(".entry > .derived")
    if derived:
        de_e = derived[0].cssselect(".de_e")
        de_c = derived[0].cssselect(".de_c")
        a_links = derived[0].cssselect("a")
        cn_def = ""
        if de_e and de_c:
            cn_def = (
                f"{de_e[0].text_content().strip()} "
                f"{de_c[0].text_content().strip()}"
            )
        cross_ref = a_links[0].text_content().strip() if a_links else ""
        return [
            {
                "word": word,
                "pos": "",
                "cn_definition": cn_def,
                "cross_ref": cross_ref,
                "sense_order": 1,
                "examples": [],
                "pronunciation": None,
                "extra_notes": None,
            }
        ]

    return []


def _oxford_pos_for_ng(n_g, base_pos_spans):
    base = " ".join("".join(s.itertext()).strip() for s in base_pos_spans).strip()
    gr_spans = n_g.cssselect(".gr")
    gr = " ".join(s.text_content().strip() for s in gr_spans)
    if gr:
        return f"{base} {gr}" if base else gr
    return base


def _oxford_extract_examples(n_g):
    examples = []
    for x_g in n_g.cssselect(".x-g"):
        en_span = x_g.cssselect(".x")
        cn_span = x_g.cssselect(".oalecd8e_chn")
        en_text = en_span[0].text_content().strip() if en_span else ""
        cn_text = cn_span[0].text_content().strip() if cn_span else ""
        if en_text or cn_text:
            examples.append((en_text, cn_text))
    return examples


def _oxford_parse_pg_direct(p_g, word, pos_spans, top_g=None):
    """Parse a p-g block that has .def-g directly (no .n-g wrappers).

    Yields entries. Grammar info comes from direct .gr children of p-g;
    examples are .x-g children of p-g.
    """
    entries = []

    base_pos = " ".join("".join(s.itertext()).strip() for s in pos_spans).strip()
    gr_spans = [
        c for c in p_g
        if c.tag == "span" and "gr" in (c.get("class", "") or "").split()
    ]
    gr_text = " ".join(s.text_content().strip() for s in gr_spans)
    pos = f"{base_pos} {gr_text}".strip() if gr_text else base_pos

    pron = _extract_oxford_pronunciation(p_g, top_g)

    def_g_children = [
        c for c in p_g
        if c.tag == "span" and "def-g" in (c.get("class", "") or "").split()
    ]

    x_g_children = [
        c for c in p_g
        if c.tag == "span" and "x-g" in (c.get("class", "") or "").split()
    ]

    for def_g in def_g_children:
        chn = def_g.cssselect(".oalecd8e_chn")
        cn_def = chn[0].text_content().strip() if chn else ""

        examples = []
        for x_g in x_g_children:
            en_span = x_g.cssselect(".x")
            cn_span = x_g.cssselect(".oalecd8e_chn")
            en_text = en_span[0].text_content().strip() if en_span else ""
            cn_text = cn_span[0].text_content().strip() if cn_span else ""
            if en_text or cn_text:
                examples.append((en_text, cn_text))

        entries.append(
            {
                "word": word,
                "pos": pos,
                "cn_definition": cn_def,
                "cross_ref": None,
                "sense_order": len(
                    [e for e in entries if e["pos"] == pos]
                )
                + 1,
                "examples": examples,
                "pronunciation": pron,
                "extra_notes": None,
            }
        )

    return entries


def _oxford_make_pos(base_pos_spans, container_el):
    """Build POS string from base pos spans and optional .gr spans in container."""
    base = " ".join(
        "".join(s.itertext()).strip() for s in base_pos_spans
    ).strip()
    gr_spans = container_el.cssselect(".gr")
    gr = " ".join(s.text_content().strip() for s in gr_spans)
    if gr:
        return f"{base} {gr}" if base else gr
    return base


def _oxford_parse_hg_direct(h_g, word, base_pos_spans, top_g):
    """Parse Pattern 4: h-g block with def-g / ids-g / x-g as direct children.

    Handles entries where .def-g sits directly under .h-g (no .p-g or .n-g
    wrappers), and entries with .ids-g / .xr-g siblings.
    """
    entries = []
    pron = _extract_oxford_pronunciation(h_g, top_g)

    # Pattern 4a: direct def-g under h-g
    def_gs = [
        c for c in h_g
        if c.tag == "span" and "def-g" in (c.get("class", "") or "").split()
    ]
    x_gs_in_hg = h_g.cssselect(".x-g")

    for def_g in def_gs:
        chn = def_g.cssselect(".oalecd8e_chn")
        cn_def = chn[0].text_content().strip() if chn else ""
        if not cn_def:
            continue

        pos = _oxford_make_pos(base_pos_spans, h_g)

        examples = []
        for x_g in x_gs_in_hg:
            # Skip x-g nested inside ids-g or infl
            skip = False
            for anc in x_g.iterancestors():
                cls = (anc.get("class", "") or "").split()
                if "ids-g" in cls or "infl" in cls:
                    skip = True
                    break
            if skip:
                continue
            en_span = x_g.cssselect(".x")
            cn_span = x_g.cssselect(".oalecd8e_chn")
            en_text = en_span[0].text_content().strip() if en_span else ""
            cn_text = cn_span[0].text_content().strip() if cn_span else ""
            if en_text or cn_text:
                examples.append((en_text, cn_text))

        entries.append(
            {
                "word": word,
                "pos": pos,
                "cn_definition": cn_def,
                "cross_ref": None,
                "sense_order": len(
                    [e for e in entries if e["pos"] == pos]
                )
                + 1,
                "examples": examples,
                "pronunciation": pron,
                "extra_notes": None,
            }
        )

    # Pattern 4b: ids-g (idioms) — each id-g becomes a separate entry
    ids_g = h_g.cssselect(".ids-g")
    for ids_container in ids_g:
        for id_g in ids_container.cssselect(".id-g"):
            id_span = id_g.cssselect(".id")
            idiom_phrase = (
                id_span[0].text_content().strip() if id_span else ""
            )

            sense_g = id_g.cssselect(".sense-g")
            if not sense_g:
                continue
            sense_g = sense_g[0]

            chn = sense_g.cssselect(".def-g .oalecd8e_chn")
            cn_def = chn[0].text_content().strip() if chn else ""
            if not cn_def:
                continue

            examples = []
            for x_g in sense_g.cssselect(".x-g"):
                en_span = x_g.cssselect(".x")
                cn_span = x_g.cssselect(".oalecd8e_chn")
                en_text = en_span[0].text_content().strip() if en_span else ""
                cn_text = (
                    cn_span[0].text_content().strip() if cn_span else ""
                )
                if en_text or cn_text:
                    examples.append((en_text, cn_text))

            entries.append(
                {
                    "word": word,
                    "pos": f"IDM {idiom_phrase}" if idiom_phrase else "IDM",
                    "cn_definition": cn_def,
                    "cross_ref": None,
                    "sense_order": len(
                        [e for e in entries if e["pos"] == f"IDM {idiom_phrase}"]
                    )
                    + 1,
                    "examples": examples,
                    "pronunciation": pron,
                    "extra_notes": None,
                }
            )

    return entries


def parse_oxford_regular(tree, word):
    entries = []
    entry_el = tree.cssselect(".entry")
    if not entry_el:
        return entries
    entry_el = entry_el[0]

    top_gs = entry_el.cssselect(".top-g")
    top_g = top_gs[0] if top_gs else None

    # Pattern 1: p-g blocks under entry (e.g. "water")
    p_gs = list(child_elements(entry_el, class_contains="p-g"))

    if p_gs:
        for p_g in p_gs:
            pos_spans = p_g.cssselect(".pos")
            if not pos_spans:
                continue

            pron = _extract_oxford_pronunciation(p_g, top_g)

            n_gs = p_g.cssselect(".n-g")
            if n_gs:
                for n_g in n_gs:
                    chn = n_g.cssselect(".def-g .oalecd8e_chn")
                    cn_def = chn[0].text_content().strip() if chn else ""

                    pos = _oxford_pos_for_ng(n_g, pos_spans)
                    entries.append(
                        {
                            "word": word,
                            "pos": pos,
                            "cn_definition": cn_def,
                            "cross_ref": None,
                            "sense_order": len(
                                [e for e in entries if e["pos"] == pos]
                            )
                            + 1,
                            "examples": _oxford_extract_examples(n_g),
                            "pronunciation": pron,
                            "extra_notes": None,
                        }
                    )
            else:
                # No .n-g — def-g and x-g are direct children of p-g
                # (e.g. "cause" verb, "abandon" noun, "above" adj)
                entries += _oxford_parse_pg_direct(p_g, word, pos_spans, top_g)
    else:
        # Pattern 2: n-g directly in h-g, POS from top-g > block-g (e.g. "beauty")
        pos_spans = tree.cssselect(".top-g .block-g .pos")
        h_g = tree.cssselect(".h-g")
        if not h_g:
            return entries
        h_g = h_g[0]

        pron = _extract_oxford_pronunciation(top_g) if top_g is not None else None

        n_gs = list(child_elements(h_g, class_contains="n-g"))
        if n_gs:
            for n_g in n_gs:
                chn = n_g.cssselect(".def-g .oalecd8e_chn")
                cn_def = chn[0].text_content().strip() if chn else ""

                pos = _oxford_pos_for_ng(n_g, pos_spans)
                entries.append(
                    {
                        "word": word,
                        "pos": pos,
                        "cn_definition": cn_def,
                        "cross_ref": None,
                        "sense_order": len(
                            [e for e in entries if e["pos"] == pos]
                        )
                        + 1,
                        "examples": _oxford_extract_examples(n_g),
                        "pronunciation": pron,
                        "extra_notes": None,
                    }
                )
        else:
            # Pattern 4: No .n-g — def-g / ids-g / x-g are direct children of h-g
            entries += _oxford_parse_hg_direct(h_g, word, pos_spans, top_g)

    # Filter out entries with no definition and no examples
    # (e.g. abbreviation expansions with no Chinese translation)
    return [e for e in entries if e["cn_definition"] or e["examples"]]


def is_oxford_xref(tree):
    return (
        bool(tree.cssselect(".sense-g .xr-g"))
        and not bool(tree.cssselect(".p-g"))
        and not bool(tree.cssselect(".n-g"))
    ) or (
        bool(tree.cssselect(".entry > .derived"))
        and not bool(tree.cssselect(".p-g"))
        and not bool(tree.cssselect(".n-g"))
    )


_COLLINS_XREF_PATTERNS = [
    "past tense of",
    "past participle of",
    "plural of",
    "plural form of",
    "present participle of",
    "spoken form of",
    "short for",
    "means the same as",
    "another form of",
    "a past tense of",
    "→see",
]


def is_collins_xref(tree):
    if bool(tree.cssselect(".st")) or bool(tree.cssselect("li")):
        return False
    captions = tree.cssselect(".caption")
    if not captions:
        return False
    caption_lower = captions[0].text_content().strip().lower()
    return any(p in caption_lower for p in _COLLINS_XREF_PATTERNS)


def parse_tabfile(filepath, source):
    """Parse a tabfile, return (entry_rows, example_batches)."""
    entry_rows = []
    example_batches = []

    with open(filepath, encoding="utf-8") as f:
        for lineno, line in enumerate(f, 1):
            if lineno % 5000 == 0:
                print(f"  {source}: {lineno} lines", file=sys.stderr)

            parts = line.split("\t", 1)
            if len(parts) < 2:
                continue
            word = parts[0].split("|")[0].strip()
            html_text = parts[1]

            try:
                tree = html.fromstring(html_text)
            except Exception:
                continue

            # Prefer word_key from HTML (handles Collins entries like "'cause"
            # whose tabfile key has a leading apostrophe while the HTML has "cause")
            if source == "collins":
                wk_els = tree.cssselect(".word_key")
                if wk_els:
                    word = wk_els[0].text_content().strip()

            if source == "collins":
                if is_collins_xref(tree):
                    entries = parse_collins_xref(tree, word)
                else:
                    entries = parse_collins_regular(tree, word)
                # Apply word-level pronunciation to all Collins entries
                pron = _extract_collins_pronunciation(tree)
                for e in entries:
                    e["pronunciation"] = pron
            else:
                if is_oxford_xref(tree):
                    entries = parse_oxford_xref(tree, word)
                else:
                    entries = parse_oxford_regular(tree, word)

            for entry in entries:
                entry_rows.append(
                    (
                        entry["word"],
                        entry["pos"],
                        entry["cn_definition"],
                        entry["cross_ref"],
                        entry["sense_order"],
                        entry["pronunciation"],
                        entry.get("extra_notes"),
                    )
                )
                example_batches.append(
                    [
                        (en, cn, i + 1)
                        for i, (en, cn) in enumerate(entry["examples"])
                    ]
                )

    return entry_rows, example_batches


def build_db(db_path, dicts_to_process, dict_paths):
    schema = Path(SCHEMA_SQL).read_text(encoding="utf-8")

    db_path = os.path.abspath(db_path)
    if os.path.exists(db_path):
        os.remove(db_path)

    conn = sqlite3.connect(db_path)
    conn.executescript(schema)
    conn.commit()

    with tempfile.TemporaryDirectory() as tmpdir:
        for source in dicts_to_process:
            dict_path = dict_paths[source]
            tabfile = os.path.join(tmpdir, f"{source}.txt")

            print(f"Extracting {source} dictionary...", file=sys.stderr)
            extract_tabfile(dict_path, tabfile)

            print(f"Parsing {source} entries...", file=sys.stderr)
            entry_rows, example_batches = parse_tabfile(tabfile, source)

            table = f"{source}_entries"
            print(
                f"Inserting {len(entry_rows)} entries into {table}...",
                file=sys.stderr,
            )

            # Insert entries and track assigned IDs
            conn.execute("BEGIN")
            entry_ids = []
            for row in entry_rows:
                cur = conn.execute(
                    f"INSERT INTO {table} (word, pos, cn_definition, cross_ref, sense_order, pronunciation, extra_notes) VALUES (?, ?, ?, ?, ?, ?, ?)",
                    row,
                )
                entry_ids.append(cur.lastrowid)
            conn.commit()

            # Build example rows with correct entry_id
            example_table = f"{source}_examples"
            example_rows = []
            for entry_id, batch in zip(entry_ids, example_batches):
                for en_text, cn_text, order in batch:
                    example_rows.append((entry_id, en_text, cn_text, order))

            print(
                f"Inserting {len(example_rows)} examples into {example_table}...",
                file=sys.stderr,
            )
            conn.execute("BEGIN")
            conn.executemany(
                f"INSERT INTO {example_table} (entry_id, en_example, cn_example, example_order) VALUES (?, ?, ?, ?)",
                example_rows,
            )
            conn.commit()

    # Populate FTS5
    print("Populating FTS5 index...", file=sys.stderr)
    for source in dicts_to_process:
        conn.execute(
            f"""
            INSERT INTO entries_fts (source, word, cn_definition, en_example)
            SELECT '{source}', e.word, e.cn_definition, x.en_example
            FROM {source}_entries e
            LEFT JOIN {source}_examples x ON e.id = x.entry_id
            WHERE e.cross_ref IS NULL OR e.cross_ref = ''
            """
        )
    conn.commit()

    # Report
    for source in dicts_to_process:
        total = conn.execute(
            f"SELECT COUNT(*) FROM {source}_entries"
        ).fetchone()[0]
        xref = conn.execute(
            f"SELECT COUNT(*) FROM {source}_entries WHERE cross_ref IS NOT NULL AND cross_ref != ''"
        ).fetchone()[0]
        examples = conn.execute(
            f"SELECT COUNT(*) FROM {source}_examples"
        ).fetchone()[0]
        print(f"  {source}: {total} entries ({xref} xref), {examples} examples")

    conn.close()
    print(f"Done: {db_path}", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser(
        description="Build ecd.db from Apple Dictionary bundles"
    )
    parser.add_argument("--output", default=DEFAULT_DB, help="Output database path")
    parser.add_argument(
        "--only",
        choices=["collins", "oxford"],
        help="Process only one dictionary",
    )
    parser.add_argument(
        "--collins-path",
        default=DEFAULT_COLLINS,
        help="Path to Collins .dictionary bundle",
    )
    parser.add_argument(
        "--oxford-path",
        default=DEFAULT_OXFORD,
        help="Path to Oxford .dictionary bundle",
    )
    args = parser.parse_args()

    dicts = [args.only] if args.only else ["collins", "oxford"]
    dict_paths = {
        "collins": args.collins_path,
        "oxford": args.oxford_path,
    }
    build_db(args.output, dicts, dict_paths)


if __name__ == "__main__":
    main()
