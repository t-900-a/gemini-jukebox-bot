#!/bin/bash
pidof  monero-wallet-rpc > /dev/null
if [[ $? -ne 0 ]] ; then
        echo "Restarting monero wallet rpc:     $(date)" >> /home/<USER>/rpc_babysitter.log
        cd /home/<USER>/public_gmi/gemlog
        /home/<USER>/monero/monero-wallet-rpc --rpc-bind-port 8374 --restricted-rpc --trusted-daemon --daemon-address <REMOTE_DAEMON> --wallet-file /home/<USER>/mywallet --tx-notify "/home/<USER>/gemini-jukebox-bot gemini://jukebox.<YOURDOMAIN>.com https://jukebox.<YOURDOMAIN>.com:8000/mpd.m3u %s monero:<YOUR_MONERO_ADDRESS> <YOUR_MONERO_SECRET_VIEW_KEY>" --password "<PASSWORD>"
fi