#!/usr/bin/env python3

import csv
import sys
from datetime import datetime

import requests


def main() -> int:
    s = requests.Session()
    with open(sys.argv[1], "r") as csvfile:
        reader = csv.DictReader(
            csvfile,
            fieldnames=["date", "time", "name", "access_granted"],
        )
        for row in reader:
            try:
                timestamp = datetime.strptime(
                    f"{row['date']} {row['time']}",
                    "%m/%d/%Y %H:%M:%S",
                )
            except ValueError:
                print("error parsing time: " + str(row), file=sys.stderr)
                continue
            r = s.post(
                "http://fcfl-access:8080/doord",
                json={
                    "timestamp": timestamp.isoformat(),
                    "name": row["name"],
                    "access_granted": row["access_granted"] == "1",
                },
            )
            print(timestamp.isoformat(), file=sys.stderr)
            r.raise_for_status()

    return 0


if __name__ == "__main__":
    sys.exit(main())
