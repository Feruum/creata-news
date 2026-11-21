Original migration references only. The application no longer imports `01.json`
or `01-source.json`. Issue metadata, HTML, photographs, videos and PDF are imported
from `backend/creata/content` into PostgreSQL and API/S3 media storage.

The existing renderer and stylesheet remain application code. Old image URLs
are routed to backend-managed media. Local legacy copies are retained as backup.
