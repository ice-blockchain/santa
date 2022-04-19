# Santa Service

``Santa is handling everything related to prizes, achievements or collectibles that the users earn through various actions.``

<details>
<summary>ACHIEVEMENTS</summary>

```
# Achievement Dictionary
## Tasks - Completed only in a specific order
1. Claim your nickname
2. Start mining
3. Upload profile picture
4. Join Telegram
5. Follow us on Twitter
6. Invite 5 friends
7. Social share
## Levels - Any of the following increases the level
1. First mining
2. 5 consecutive mining sessions ( no more than 10 hours pause between the mining sessions )
3. 10 consecutive mining sessions
4. 30 consecutive mining sessions
5. 60 consecutive mining sessions
6. 90 consecutive mining sessions
7. Completed tasks from home screen ( each task 1 level )
8. Confirm phone number
9. 1 Friend from agenda joined ICE
10. 5 Friends from agenda joined ICE
11. 10 Friends from agenda joined ICE
12. Opened the app 5 times
13. Opened the app 10 times
14. Opened the app more than 30 times in the last 30 days
15. First ping to referral
16. 10 pings to referrals
## Badges:
### Levels:
1. Level 1
2. Level 5
3. Level 10
4. Level 15
5. Level 20
### Coins (ICE Balance):
1. 0-1000
2. 1001-5000
3. 5001-10000
4. 10001-40000
5. 40001-80000
6. 80001-160000
7. 160001-320000
8. 320001-640000
9. 640001-1280000
10. 1280001+
### Social:
1. 0-5 referrals
2. 6-15 referrals
3. 16-30 referrals
4. 31-100 referrals
5. 101-250 referrals
6. 251-500 referrals
7. 501-1000 referrals
8. 1001-2000 referrals
9. 2001-10000 referrals
10. 10001+ referrals
```
</details>

### Development

These are the crucial/critical operations you will need when developing `Santa`:

1. If you need a DID token for `magic.link` for testing locally, see https://github.com/magiclabs/magic-admin-go/blob/master/token/did_test.go
2. `make run`
    1. This runs the actual service.
    2. It will feed off of the properties in `./application.yaml`
    3. By default, https://localhost/achievements runs the Open API (Swagger) entrypoint.
3. `make start-test-environment`
    1. This bootstraps a local test environment with **Santa**'s dependencies using your `docker` and `docker-compose` daemons.
    2. It is a blocking operation, SIGTERM or SIGINT will kill it.
    3. It will feed off of the properties in `./application.yaml`
        1. MessageBroker GUIs
            1. https://www.conduktor.io
            2. https://www.kafkatool.com
            3. (CLI) https://vectorized.io/redpanda
        2. DB GUIs
            1. https://github.com/tarantool/awesome-tarantool#gui-clients
            2. (CLI) `docker exec -t -i mytarantool console` where `mytarantool` is the container name
4. `make all`
    1. This runs the CI pipeline, locally -- the same pipeline that PR checks run.
    2. Run it before you commit to save time & not wait for PR check to fail remotely.
5. `make local`
    1. This runs the CI pipeline, in a descriptive/debug mode. Run it before you run the "real" one.
    2. If this takes too much it might be because of `make buildMultiPlatformDockerImage` that builds docker images for `arm64 s390x ppc64le` as well. If that's
       the case, remove those 3 platforms from the iteration and retry (but don't commit!)
6. `make lint`
    1. This runs the linters. It is a part of the other pipelines, so you can run this separately to fix lint issues.
7. `make test`
    1. This runs all tests.
8. `make benchmark`
    1. This runs all benchmarks.
