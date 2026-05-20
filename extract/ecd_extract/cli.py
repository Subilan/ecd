"""CLI entry point for ecd_extract."""

import argparse

from .build import build_db
from .config import DEFAULT_COLLINS, DEFAULT_DB, DEFAULT_OXFORD


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
