# Bot for Prometheus' Alertmanager

## Installation

Edit TELEGRAM_ADMIN in docker-compose.yml file to the bot admin telegramID.

Usage within docker-compose

```bash
docker-compose build
docker-compose up -d
```

## Commands

###### /start

> Hey, Matthias! I will now keep you up to date!  
> [/help](#help)

###### /stop

> Alright, Matthias! I won't talk to you again.  
> [/help](#help)

###### /alerts

> 🔥 **FIRING** 🔥  
> **NodeDown** (Node scraper.krautreporter:8080 down)  
> scraper.krautreporter:8080 has been down for more than 1 minute.  
> **Started**: 1 week 2 days 3 hours 50 minutes 42 seconds ago  
> 
> 🔥 **FIRING** 🔥
> **monitored_service_down** (MONITORED SERVICE DOWN)
> The monitoring service 'digitalocean-exporter' is down.
> **Started**: 10 seconds ago

###### /silences

> NodeDown 🔕  
>  `job="ranch-eye" monitor="exporter-metrics" severity="page"`  
> **Started**: 1 month 1 week 5 days 8 hours 27 minutes 57 seconds ago  
> **Ends**: -11 months 2 weeks 2 days 19 hours 15 minutes 24 seconds  
> 
> RancherServiceState 🔕  
>  `job="rancher" monitor="exporter-metrics" name="scraper" rancherURL="http://rancher.example.com/v1" severity="page" state="inactive"`  
> **Started**: 1 week 2 days 3 hours 46 minutes 21 seconds ago  
> **Ends**: -3 weeks 1 day 13 minutes 24 seconds  

###### /chats

> Currently these chat have subscribed:
> @MetalMatze

###### /status

> **AlertManager**  
> Version: 0.5.1  
> Uptime: 3 weeks 1 day 6 hours 15 minutes 2 seconds  
> **AlertManager Bot**  
> Version: 0.3.1  
> Uptime: 3 weeks 1 hour 17 minutes 19 seconds  

###### /help

> I'm a Prometheus AlertManager Bot for Telegram. I will notify you about alerts.  
> You can also ask me about my [/status](#status), [/alerts](#alerts) & [/silences](#silences)  
>   
> Available commands:  
> [/start](#start) - Subscribe for alerts.  
> [/stop](#stop) - Unsubscribe for alerts.  
> [/status](#status) - Print the current status.  
> [/alerts](#alerts) - List all alerts.  
> [/silences](#silences) - List all silences.  
> [/chats](#chats) - List all users and group chats that subscribed.
> [/members](#members) - List all members.
> [/addmember](#addmember) - Add a member.
> [/rmmember](#rmmember) - Remove a member.
> [/nodes](#nodes) - List all nodes.

###### /members
> Currently these members have added:
> @vulong2 level: 1
> @boss level: 3
> @cto level: 3
> @leader1 level: 2
> @leader2 level: 2
> @vu_long level: 1
> @vulong3 level: 2
> @vulong4 level: 3

###### /addmember
Right format: '/addmember username level (node if level = 1)'. Ex: /addmember vu_long 1 httpd
> /addmember vu_long 1 httpd
> /addmember vu_long2 1 nginx
> /addmember techmanager 2
> /addmember techleader 2
> /addmember SRE 3
> /addmember CTO 3
> /addmember CEO 3
> Already do your wish!

###### /rmmember
Right format: '/rmmember username'. Ex: /rmmember vu_long
> Already do your wish!

###### /nodes
> Currently these nodes have added:
> @httpd level: vu_long5
> @nginx level: vulong2

### Configuration

ENV Variable | Description
|-------------------|------------------------------------------------------|
| ALERTMANAGER_URL  | Address of the alertmanager, default: `http://localhost:9093` |
| CONSUL_URL        | The URL to use to connect with Consul, default: `localhost:8500` |
| LISTEN_ADDR       | Address that the bot listens for webhooks, default: `0.0.0.0:8080` |
| STORE             | The type of the store to use, choose from bolt (local) or consul (distributed) |
| TELEGRAM_ADMIN    | The Telegram user id for the admin. The bot will only reply to messages sent from an admin. All other messages are dropped and logged on the bot's console. |
| TELEGRAM_TOKEN    | Token you get from [@botfather](https://telegram.me/botfather) |
| TEMPLATE_PATHS    | Path to custom message templates, default template is `./default.tmpl`, in docker - `/templates/default.tmpl` |

## Development

Get all dependencies. We use [golang/dep](https://github.com/golang/dep).  
Fetch all dependencies with:

```
dep init
dep ensure -v -vendor-only
```

Build the binary using `make`:

```
make build
```

In case you have `$GOPATH/bin` in your `$PATH` you can now simply start the bot by running:

```bash
./alertmanager-bot
```

## Missing

##### Commands

* `/silence` - show a specific silence  
* `/silence_del` - delete a silence by command  
* `/silence_add` - add a silence for a alert by command
