# str3k0za radar

This is a Telegram bot on Go that acts as an automated cybersecurity advisor, delivering digests of MITRE ATT&CK techniques (TTPs) and Common Weakness Enumerations (CWE), based on DFIR.

Users can finely tune their feed by filtering indicators based on specific APT groups tracked in DFIR reports. Future iterations will expand the ingestion pipeline to parse TheHackerNews, TheDFIRReport, and public Telegram channels.


Public data from:

- https://github.com/mitre/cti
- https://cwe.mitre.org/data/downloads.html (research dataset)
