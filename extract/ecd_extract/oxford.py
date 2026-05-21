"""Oxford Advanced Learner's dictionary parser."""

import json
import re

from .utils import child_elements, clean_ipa


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
        text = clean_ipa(phon.text_content())
        if text:
            ipas.append(text)
    return json.dumps(ipas, ensure_ascii=False) if ipas else None


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
        ], [[]], [[]], [], []

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
        ], [[]], [[]], [], []

    return [], [[]], [[]], [], []


def _oxford_pos_for_ng(n_g, base_pos_spans):
    base = " ".join("".join(s.itertext()).strip() for s in base_pos_spans).strip()
    gr_spans = n_g.cssselect(".gr")
    gr = " ".join(s.text_content().strip() for s in gr_spans)
    if gr:
        return f"{base} {gr}" if base else gr
    return base


def _oxford_extract_cn_def(def_g):
    """Extract Chinese definition from a def-g element.

    Prefers .oalecd8e_chn inside .d or .ud (actual definitions) over
    those inside .label-g (usage notes / grammar labels).
    """
    cn_parts = []
    for d_el in def_g.cssselect(".d, .ud"):
        for chn in d_el.cssselect(".oalecd8e_chn"):
            text = chn.text_content().strip()
            if text:
                cn_parts.append(text)
    if cn_parts:
        return "；".join(cn_parts)
    chn = def_g.cssselect(".oalecd8e_chn")
    return chn[0].text_content().strip() if chn else ""


def _extract_oxford_idioms(container_el):
    """Extract idioms from .ids-g elements inside container_el.

    If container_el itself is an ids-g element, processes it directly.
    Returns (idiom_data, idiom_examples) where:
      idiom_data = [{"idiom_phrase": str, "cn_definition": str}, ...]
      idiom_examples = [[(en, cn), ...], ...] aligned with idiom_data
    """
    idiom_data = []
    idiom_examples = []
    classes = (container_el.get("class", "") or "").split()
    if "ids-g" in classes:
        ids_containers = [container_el]
    else:
        ids_containers = container_el.cssselect(".ids-g")
    for ids_container in ids_containers:
        for id_g in ids_container.cssselect(".id-g"):
            id_span = id_g.cssselect(".id")
            idiom_phrase = (
                id_span[0].text_content().strip() if id_span else ""
            )

            sense_g = id_g.cssselect(".sense-g")
            if not sense_g:
                continue
            sense_g = sense_g[0]

            def_g = sense_g.cssselect(".def-g")
            cn_def = _oxford_extract_cn_def(def_g[0]) if def_g else ""
            if not cn_def and not idiom_phrase:
                continue

            examples = []
            for x_g in sense_g.cssselect(".x-g"):
                en_span = x_g.cssselect(".x")
                cn_span = x_g.cssselect(".oalecd8e_chn")
                en_text = en_span[0].text_content().strip() if en_span else ""
                cn_text = cn_span[0].text_content().strip() if cn_span else ""
                if en_text or cn_text:
                    examples.append((en_text, cn_text))

            idiom_data.append({
                "idiom_phrase": idiom_phrase,
                "cn_definition": cn_def,
            })
            idiom_examples.append(examples)
    return idiom_data, idiom_examples


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
        cn_def = _oxford_extract_cn_def(def_g)

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
    entry_synonyms = []
    entry_antonyms = []
    pron = _extract_oxford_pronunciation(h_g, top_g)

    # Pattern 4a: direct def-g under h-g
    def_gs = [
        c for c in h_g
        if c.tag == "span" and "def-g" in (c.get("class", "") or "").split()
    ]
    x_gs_in_hg = h_g.cssselect(".x-g")

    for def_g in def_gs:
        cn_def = _oxford_extract_cn_def(def_g)
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
        # Extract xrefs from h_g for this entry (shared across 4a entries)
        syns, ants = _extract_oxford_xrefs(h_g)
        entry_synonyms.append(syns)
        entry_antonyms.append(ants)

    # Pattern 4b: ids-g (idioms) — return as separate idiom data
    hg_idiom_data, hg_idiom_examples = _extract_oxford_idioms(h_g)

    return entries, entry_synonyms, entry_antonyms, hg_idiom_data, hg_idiom_examples


def _extract_oxford_xrefs(container_el):
    """Extract synonym and antonym cross-references from Oxford .xr-g elements."""
    synonyms = []
    antonyms = []
    for xr_g in container_el.cssselect(".xr-g"):
        opp = xr_g.cssselect(".symbols-oppsym")
        if opp:
            xr = xr_g.cssselect(".xr .xh a")
            if xr:
                w = xr[0].text_content().strip()
                if w:
                    antonyms.append(w)
            continue
        syn = xr_g.cssselect(".symbols-synsym")
        if syn:
            xr = xr_g.cssselect(".xr .xh a")
            if xr:
                w = xr[0].text_content().strip()
                if w:
                    synonyms.append(w)
            continue
        z_xr = xr_g.cssselect(".z_xr")
        if z_xr and "synonym" in z_xr[0].text_content().lower():
            xr = xr_g.cssselect(".xr .xh a")
            if xr:
                w = xr[0].text_content().strip()
                if w:
                    synonyms.append(w)
    return synonyms, antonyms


def parse_oxford_regular(tree, word):
    entries = []
    entry_synonyms = []
    entry_antonyms = []
    all_idiom_data = []
    all_idiom_examples = []
    entry_el = tree.cssselect(".entry")
    if not entry_el:
        return [], [[]], [[]], [], []
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

            p_g_idiom_data, p_g_idiom_examples = _extract_oxford_idioms(p_g)
            if p_g_idiom_data:
                all_idiom_data.extend(p_g_idiom_data)
                all_idiom_examples.extend(p_g_idiom_examples)

            n_gs = p_g.cssselect(".n-g")
            if n_gs:
                for n_g in n_gs:
                    def_g = n_g.cssselect(".def-g")
                    cn_def = _oxford_extract_cn_def(def_g[0]) if def_g else ""

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
                    syns, ants = _extract_oxford_xrefs(n_g)
                    entry_synonyms.append(syns)
                    entry_antonyms.append(ants)
            else:
                # No .n-g — def-g and x-g are direct children of p-g
                # (e.g. "cause" verb, "abandon" noun, "above" adj)
                pg_entries = _oxford_parse_pg_direct(p_g, word, pos_spans, top_g)
                entries += pg_entries
                syns, ants = _extract_oxford_xrefs(p_g)
                for _ in pg_entries:
                    entry_synonyms.append(syns)
                    entry_antonyms.append(ants)
    else:
        # Pattern 2: n-g directly in h-g, POS from top-g > block-g (e.g. "beauty")
        pos_spans = tree.cssselect(".top-g .block-g .pos")
        # Fallback for run-on entries (e.g. "radically"): POS in .top-g > .pos-g > .pos
        if not pos_spans:
            pos_spans = tree.cssselect(".top-g > .pos-g .pos")
        h_g = tree.cssselect(".h-g")
        if not h_g:
            return [], [[]], [[]], [], []
        h_g = h_g[0]

        pron = _extract_oxford_pronunciation(top_g) if top_g is not None else None

        n_gs = list(child_elements(h_g, class_contains="n-g"))
        if n_gs:
            for n_g in n_gs:
                def_g = n_g.cssselect(".def-g")
                cn_def = _oxford_extract_cn_def(def_g[0]) if def_g else ""

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
                syns, ants = _extract_oxford_xrefs(n_g)
                entry_synonyms.append(syns)
                entry_antonyms.append(ants)
        else:
            # Pattern 4: No .n-g — def-g / ids-g / x-g are direct children of h-g
            hg_entries, hg_syns, hg_ants, hg_idiom_data, hg_idiom_examples = (
                _oxford_parse_hg_direct(h_g, word, pos_spans, top_g)
            )
            entries += hg_entries
            entry_synonyms += hg_syns
            entry_antonyms += hg_ants
            if hg_idiom_data:
                all_idiom_data.extend(hg_idiom_data)
                all_idiom_examples.extend(hg_idiom_examples)

    # Entry-level idioms: ids-g as direct children of .entry (outside h-g/p-g)
    entry_ids_children = [
        c for c in entry_el
        if c.tag == "span" and "ids-g" in (c.get("class", "") or "").split()
    ]
    for ids_c in entry_ids_children:
        el_idiom_data, el_idiom_examples = _extract_oxford_idioms(ids_c)
        if el_idiom_data:
            all_idiom_data.extend(el_idiom_data)
            all_idiom_examples.extend(el_idiom_examples)

    # Fill in word for all idiom entries
    for i in all_idiom_data:
        i["word"] = word

    # Words with only idioms: create minimal entry to ensure word is searchable
    if not entries and all_idiom_data:
        entries.append(
            {
                "word": word,
                "pos": "",
                "cn_definition": "",
                "cross_ref": None,
                "sense_order": 1,
                "examples": [],
                "pronunciation": (
                    _extract_oxford_pronunciation(top_g)
                    if top_g is not None
                    else None
                ),
                "extra_notes": None,
                "_idiom_host": True,
            }
        )
        entry_synonyms.append([])
        entry_antonyms.append([])

    # Filter out entries with no definition and no examples
    # (e.g. abbreviation expansions with no Chinese translation)
    # Keep idiom-only entries that serve as hosts for /idm lookup
    filtered = [
        e for e in entries
        if e["cn_definition"] or e["examples"] or e.get("_idiom_host")
    ]

    # Clean up internal marker
    for e in filtered:
        e.pop("_idiom_host", None)

    # Filter synonym/antonym lists to match filtered entries
    filtered_syns = [
        s for e, s in zip(entries, entry_synonyms)
        if e["cn_definition"] or e["examples"] or e.get("_idiom_host")
    ]
    filtered_ants = [
        a for e, a in zip(entries, entry_antonyms)
        if e["cn_definition"] or e["examples"] or e.get("_idiom_host")
    ]

    return filtered, filtered_syns, filtered_ants, all_idiom_data, all_idiom_examples
