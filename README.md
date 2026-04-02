# remotepi

Control a Marantz PM6007 stereo receiver and LED strip via IR/wired signals, integrated with Home Assistant through MQTT.

## Features

- **Marantz Control**: Sends ON/OFF signals to Marantz receivers when ALSA audio streams start/stop
- **LED Strip Control**: Control RGBW LED strips via IR through Home Assistant MQTT
- **Home Assistant Integration**: Automatic MQTT discovery for LED strip

## Hardware Setup

- **GPIO 27 (Pin 13)**: Marantz wired remote control
- **GPIO 17 (Pin 11)**: LED strip IR transmitter

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MQTT_BROKER` | `tcp://localhost:1883` | MQTT broker URL |
| `MQTT_CLIENT_ID` | `remotepi` | MQTT client identifier |
| `MQTT_USERNAME` | *(empty)* | MQTT username (optional) |
| `MQTT_PASSWORD` | *(empty)* | MQTT password (optional) |
| `MARANTZ_PIN` | `27` | GPIO pin for Marantz control |
| `LED_PIN` | `17` | GPIO pin for LED IR transmitter |
| `OFF_DELAY` | `2m` | Delay before turning off Marantz |
| `AUDIO_CARD` | `card2` | ALSA audio card to monitor |

## Installation

### Systemd Service (LibreELEC/Raspberry Pi OS)

```bash
cat <<EOF > /storage/.config/system.d/remotepi.service
[Unit]
Description=RemotePi Service - Marantz and LED Control
After=network.target

[Service]
Type=simple
Environment=MQTT_BROKER=tcp://192.168.1.10:1883
Environment=MQTT_USERNAME=your_username
Environment=MQTT_PASSWORD=your_password
Environment=MARANTZ_PIN=27
Environment=LED_PIN=17
Environment=OFF_DELAY=2m
Environment=AUDIO_CARD=card2
ExecStart=/storage/.kodi/remotepi
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable remotepi
systemctl start remotepi
```

## Home Assistant Integration

The LED strip appears automatically in Home Assistant via MQTT discovery.

MQTT Topics:
- **Discovery**: `homeassistant/light/remotepi_ledstrip/config`
- **Command**: `homeassistant/light/remotepi_ledstrip/set`
- **State**: `homeassistant/light/remotepi_ledstrip/state`

Example command payload:
```json
{
  "state": "ON",
  "brightness": 200,
  "color": {"r": 255, "g": 0, "b": 0},
  "effect": "none",
  "transition": 2
}
```

Supported effects: `none`, `flash`, `fade`, `smooth`, `strobe`

### Transitions

The `transition` parameter (in seconds) enables smooth brightness changes:
- HA will send `transition: 2` for a 2-second fade
- Commands are queued and spaced 100ms apart
- Brightness changes are spread over the transition duration

Without `transition`, brightness changes are instant (within IR processing limits).

### Brightness Step (Dimmer Support)

For rotary dimmers or hold-to-dim switches (e.g., EnOcean Friends of Hue), use `brightness_step`:

```json
{
  "state": "ON",
  "brightness_step": 25
}
```

- Positive values increase brightness
- Negative values decrease brightness
- Optimized for rapid events: sends 1-5 IR commands per step
- No queue backup: immediate response per event

### EnOcean/Zigbee Dimmer Integration

For EnOcean Friends of Hue switches via Zigbee2MQTT, create an automation in Home Assistant:

```yaml
automation:
  - alias: "LED Strip Dimmer"
    trigger:
      - platform: mqtt
        topic: "zigbee2mqtt/your_switch"
    action:
      - service: mqtt.publish
        data:
          topic: "homeassistant/light/remotepi_ledstrip/set"
          payload: >
            {% if trigger.payload_json.action == 'brightness_move_up' %}
            {"state": "ON", "brightness_step": 20}
            {% elif trigger.payload_json.action == 'brightness_move_down' %}
            {"state": "ON", "brightness_step": -20}
            {% elif trigger.payload_json.action == 'brightness_stop' %}
            {}
            {% elif trigger.payload_json.action == 'toggle' %}
            {"state": "ON"}
            {% endif %}
```

## LED Color Mapping

Since the LED controller only has preset colors, RGB values are mapped to the nearest available preset:
- Red, Green, Blue, White
- 12 additional color presets

Brightness is controlled by sending BRIGHTNESSUP/DOWN commands based on the difference from current level.

## Build

```bash
go build -o remotepi .
```

## Dependencies

- `github.com/stianeikeland/go-rpio/v4` - Raspberry Pi GPIO
- `github.com/morphar/powernap` - Precise timing for IR signals
- `github.com/eclipse/paho.mqtt.golang` - MQTT client
