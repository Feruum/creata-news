"""Operator-only interactive collector login. Never exposes the session in logs."""
import asyncio
import os
from pathlib import Path

async def main():
    from telethon import TelegramClient
    from telethon.sessions import StringSession
    api_id=int(os.environ['TELEGRAM_API_ID']);api_hash=os.environ['TELEGRAM_API_HASH']
    target=Path(os.getenv('TELEGRAM_COLLECTOR_SESSION_FILE','../.runtime/telegram-collector.session')).resolve()
    target.parent.mkdir(parents=True,exist_ok=True)
    client=TelegramClient(StringSession(),api_id,api_hash)
    await client.start()
    try:
        descriptor=os.open(target,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
        with os.fdopen(descriptor,'w') as file:file.write(client.session.save())
        print('Collector session saved to '+str(target)+'. Keep this file private.')
    finally:await client.disconnect()

if __name__=='__main__':asyncio.run(main())
