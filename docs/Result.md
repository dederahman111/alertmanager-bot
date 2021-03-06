# Alerting System

## Specifications:

### 1. Node Exporter:
- Running an exporter agent to collect service/server metrics and show via
HTTP/HTTPS endpoint.
RESULT: Handle monitor HTTPD and NGINX services.

### 2. Prometheus:
- Automatically scrape metrics from Node Exporter after an interval time.
- Set rules for alerting.
RESULT: Automatically scrape metrics from Node Exporter after an interval time.
NOT DONE: PUSH VIA WEBHOOK.

### 3. Alertmanager:
- Receiving signal from Prometheus and forward the alert to Telegram Bot.
RESULT: DONE but can not run inside a docker container.
ERROR: ...

### 4. Levels of alerting:
- Define 3 levels of alert
    + Level 1: owners of each service (or server)
    + Level 2: Tech Manager
    + Level 3: SRE
- When alert is sent, each level need to do response in 5 minutes by one of 2
actions:
    + Acknowledge
    + Forward
- If there is no any response in 5 minutes, the alert will be sent to next levels.
RESULT: DONE.

### 5. Database:
- Storing service/server name, owners with different levels, Telegram ID of each
person.
RESULT: DONE.

### 6. Telegram Chat Group:
- A Telegram chat group that including all Tech owners. Everyone disable
notification of that group by default, but not tagging.
RESULT: DONE.

### 7. Telegram Bot:
Description: Handle the alerts from Alertmanager and send to Telegram chat group.
- Receiving alert/resolved messages from AlertManager.
- Handling messages:
    + The alert message will be send to Telegram Group with tag a TelegramID for level 1 recipients (ex: wukong - owner in below photo) and two buttons: *'Acknowledge'*, *'Forward'*.
    + If anyone in chat group click to _'Acknowledge'_ button, then:
        
        ● Hide all buttons.
        
        ● Show username who did acknowledge the alert.
        
        ● Stop auto forwarding to next Level recipients.
    + If anyone in chat group click to ‘Forward’ button, then:
        
        ● Show username that did forward the alert and username of Level 2 of recipients.
        
        ● Button ‘Forward’ will be hide.
    + If no one action that message in 5 minutes, then:
        
        ● The message will be auto forward for next level (Level 2).
        
        ● When the highest level was forwarded, the button *‘Forward’* will be hide.
    + If the bot received a resolved message, then:
        
        ● Hide all buttons of previous alert message.
        
        ● Stop auto forward to next Level of previous alert message.

RESULT: DONE.