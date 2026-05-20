"""Collins COBUILD dictionary parser."""

import copy
import json
import re

from .utils import clean_ipa, itertext


def _extract_collins_pronunciation(tree):
    """Extract pronunciation from Collins word_entry. Returns JSON string or None."""
    pron_el = tree.cssselect(".word_entry > .pron")
    if not pron_el:
        return None
    ipas = []
    for type_el in pron_el[0].cssselect(".type_uk, .type_us"):
        text = clean_ipa(type_el.text_content())
        if text:
            ipas.append(text)
    return json.dumps(ipas, ensure_ascii=False) if ipas else None


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
    ], [[]]


def _extract_notes_from_figures(figures):
    """Extract extra_notes from figure.note elements. Returns list of note dicts.

    Each figure produces exactly one note, combining any inline definition text
    with its <li> examples so they display as a single unit.
    """
    notes = []
    for fig in figures:
        cls = fig.get("class", "")
        note_type = ""
        for part in cls.split():
            if part.startswith("type-"):
                note_type = part.replace("type-", "")
                break

        # Quotation figures: parse .cit elements individually
        if note_type == "quotation":
            cits = fig.cssselect(".cit")
            if cits:
                cit_parts = []
                for cit in cits:
                    # Quote text from blockquote > p or span.quote
                    bq = cit.cssselect("blockquote p")
                    sq = cit.cssselect("span.quote")
                    if bq:
                        q_text = itertext(bq[0])
                    elif sq:
                        # Replace <br> with "\n" text nodes
                        for br in sq[0].cssselect("br"):
                            br.tail = ("\n" + (br.tail or ""))
                        raw = sq[0].text_content()
                        q_text = re.sub(r"[ \t]+", " ", raw).strip()
                    else:
                        q_text = ""
                    # Attribution from cite
                    cite_el = cit.cssselect("cite")
                    attr = cite_el[0].text_content().strip() if cite_el else ""
                    if q_text and attr:
                        cit_parts.append(f"{q_text}\n{attr}")
                    elif q_text:
                        cit_parts.append(q_text)

                if cit_parts:
                    final_en = "\n\n".join(cit_parts)
                    notes.append(
                        {"type": note_type, "en": final_en, "cn": ""}
                    )
                continue

        # Extract definition / explanation text (non-<ul> content).
        fig_copy = copy.deepcopy(fig)
        for ul in fig_copy.cssselect("ul"):
            parent = ul.getparent()
            if parent is not None:
                parent.remove(ul)
        def_cn_spans = fig_copy.cssselect(".def_cn")
        cn_parts = []
        for s in def_cn_spans:
            cn_parts.append(s.text_content().strip())
            parent = s.getparent()
            if parent is not None:
                parent.remove(s)
        cn_text = " ".join(p for p in cn_parts if p)
        en_text = itertext(fig_copy)
        # Strip dictionary UI labels ("Usage Note :", "Quotations ", etc.)
        en_text = re.sub(r"^(Usage Note|Quotations?)\s*:?\s*", "", en_text)

        # Collect examples from <li><p> pairs
        examples = []
        for li in fig.cssselect("li"):
            ps = li.cssselect("p")
            if len(ps) >= 2:
                ex_en = itertext(ps[0])
                ex_cn = itertext(ps[1])
                if ex_en or ex_cn:
                    examples.append((ex_en, ex_cn))

        # Combine definition and examples into a single note
        parts_en = [en_text] if en_text else []
        parts_cn = [cn_text] if cn_text else []
        for ex_en, ex_cn in examples:
            if ex_en and ex_cn:
                parts_en.append(ex_en)
                parts_cn.append(ex_cn)
            elif ex_en:
                parts_en.append(ex_en)
            elif ex_cn:
                parts_cn.append(ex_cn)

        final_en = "\n".join(parts_en).strip()
        final_cn = "\n".join(parts_cn).strip()
        if final_en or final_cn:
            notes.append(
                {"type": note_type, "en": final_en, "cn": final_cn}
            )

    return notes


def _extract_drv_entries(figures, word, pronunciation):
    """Create entry dicts from type-drv figure.note elements (derived forms)."""
    entries = []
    for fig in figures:
        b_tag = fig.cssselect("b")
        if not b_tag:
            continue
        drv_word = b_tag[0].text_content().strip()
        if not drv_word:
            continue

        examples = []
        for li in fig.cssselect("li"):
            ps = li.cssselect("p")
            if len(ps) >= 2:
                en_text = itertext(ps[0])
                cn_text = itertext(ps[1])
                if en_text or cn_text:
                    examples.append((en_text, cn_text))

        # Handle caption format (e.g. "bleakly")
        cn_def = ""
        caption = fig.cssselect(".caption")
        if caption:
            def_cn = caption[0].cssselect(".def_cn")
            if def_cn:
                cn_def = def_cn[0].text_content().strip()

        if not cn_def and not examples:
            continue

        entries.append({
            "word": drv_word,
            "pos": "DRV",
            "cn_definition": cn_def,
            "cross_ref": None,
            "sense_order": 1,
            "examples": examples,
            "pronunciation": pronunciation,
            "extra_notes": None,
        })
    return entries


def _extract_collins_synonyms(div):
    """Extract synonym words from a .collins_en_cn div's .synonym block."""
    syn_div = div.cssselect(".synonym")
    if not syn_div:
        return []
    synonyms = []
    for span in syn_div[0].cssselect("span.form"):
        a_tag = span.cssselect("a")
        if a_tag:
            synonyms.append(a_tag[0].text_content().strip())
        else:
            text = span.text_content().strip()
            if text:
                synonyms.append(text)
    return synonyms


def parse_collins_regular(tree, word):
    entries = []
    entry_synonyms = []  # list of lists, aligned with entries
    seen_drv_words = set()  # deduplicate derived words within same entry
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

        # Separate type-drv figures (derived forms) from other note figures
        all_figs = div.cssselect("figure.note")
        drv_figs = [f for f in all_figs if "type-drv" in (f.get("class", "") or "").split()]
        other_figs = [f for f in all_figs if f not in drv_figs]

        en_cn_notes = _extract_notes_from_figures(other_figs) if other_figs else []
        en_cn_notes_json = (
            json.dumps(en_cn_notes, ensure_ascii=False) if en_cn_notes else None
        )

        synonyms = _extract_collins_synonyms(div)

        # Create derived-form entries from type-drv figures (always, even
        # if the main entry below is empty)
        drv_entries = _extract_drv_entries(drv_figs, word, None)
        for de in drv_entries:
            if de["word"] not in seen_drv_words:
                seen_drv_words.add(de["word"])
                entries.append(de)
                entry_synonyms.append([])

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
        entry_synonyms.append(synonyms)

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

    return entries, entry_synonyms
