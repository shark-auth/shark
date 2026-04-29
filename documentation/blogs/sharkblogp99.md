Hey everyone! Hope everyone is okay, welcome to this shark devlog, today's topic involves one of the most embarassing moments of shark's first development cycle.

With YC approaching and other deadlines I want to attend(Including shark's delayed launch on april 29) I encountered myself in the living room, working on final landing page fixes, demo recording, blog writing and all that kind of stuff you do before a release. that's when it hit me: hey I have not tested shark's benchmarks. Earlier this week I asked Claude to develop a full Benchmark Suite for shark, purpose was to include benchmarks on readme, maybe on landing, but at the end of the week, with security fixes and UI bugs in mind, I forgot about shark's benchmarks. To anyone who does not know me or shark's dev cycle from hell. I'm 18. shipping shark between calc sessions and fixing issues at my sysadmin job. Really obsessed about making auth easily deployable and ready for the agentic future. Started to focus heavily on things like "Cool UX" ensuring UI had no big bugs, no major security issues and etc. For some single stupid reason I thought " I've written go before, I have done go projects before, go fast, why should i worry? this is not python" Big error. As I was ego tripping looking at the awesome delegation chains canvas and convincing myself I had put together one of the best auth solutions in the market (lots of work needed, I admit it, this is just launch.) I figured out i would leave benchmarks for later.

So there i was shipping the last changes and then i executed the benchmark and come across the awesome benchmark results: P99 2s, RPS 8. My jaw dropped. I couldn't believe it, seriously. I never cared to take a decision based on performance related what a surprise performance is not good!! So I looked at the code. Realized argon2 parameters were too heavy on the CPU. 3 passes, 64 MB, never really thought that a config like that maybe was not suitable for running in a $5VPS, so that's the first thing I fixed. Then, i came across the realization that I was really messing up I/O with the terminal output. Even though Shark's Public log system was okay and not harming performance, terminal log system was actually a mess. Every log printed to TTY was blocking I/O via rw mutex. I looked with horror as I just realized what i had done. So i thought it would be a great idea to dispatch GEMINI CLI to fix this, at the start, Gemini did good, it suggested a benchmark with no logs printing to tty to see "true performance", we ran it, performance was better showing 47 RPS and 480 ms p99s so i thought that maybe we could get down with a few tweaks. didn't mind it and told Gemini to get us to 300+ RPS and sub 300ms p99s. Really dumb to think this would really solve the issue. That's where we got to the worst benchmark OAT: 5s p99, 30 rps. Title reads with the worst performance achieved in both metrics. So i had to put hands on to work. Midnight, launch day, the less i wanna do is refactor stuff, but i had to. The launch is important to me and performance was too mediocre to do so. So I started investigating, first thing i looked, was at pocketbase, I had the ideas of its architecture, not the full picture, asked AI to explain to me how they achieve that good of a performance, noticed gaps in my code and similarities to theirs, based off that, first thing i did was tune the SQLite config to one writer and 15 readers. This alone brought p99 from the 5s to 2.1s and RPS TO 56-64 59.6 good start. Next thing I thought of was tweaking argon parameters again. Current config is 16 MB and one iteration, might change it after some testing, but for now this is the sweet spot for me and it enters OWASP minimum territory, won't go past that. So the obvious read was: true cache layer. Only two bottlenecks i could think of were processor abusing via hashing where we shouldn't and excessive DB calling, cache layer = fixes both. So there I started. Used Gemini CLI Again to run a codebase exploration agent to call all the redundant DB calls and hashing we can avoid via caching. After the initial caching refactor, we stopped majority of argon2 hashing ops that were redundantly ocurring and diminished db hitting by a lot. switching to a 90% reading 10% writing model. This brought benchmarks to range between 300-500 RPS on my dying laptop.

After Closing chrome, getting clutter out we got this results.

(Ignore ones in 0, they failed because of API key)

With TTY output: === summary (profile: smoke) ===

scenario | rps | p50 | p95 | p99 | err

---

signup_storm | 97.2 | 91ms | 191ms | 277ms | 0

login_burst | 285.1 | 22ms | 91ms | 177ms | 0

oauth_client_credentials | 548.9 | 8ms | 82ms | 111ms | 1

token_exchange_chain | 0.0 | 0ms | 0ms | 0ms | 1

cascade_revoke_user_agents | 0.0 | 0ms | 0ms | 0ms | 1

oauth_dpop | 211.5 | 157ms | 416ms | 1.52s | 0

rbac_permission_check_hot | 0.0 | 0ms | 0ms | 0ms | 1

vault_read_concurrent | 0.0 | 0ms | 0ms | 0ms | 1

WITHOUT TTY OUTPUT == === summary (profile: smoke) ===

scenario | rps | p50 | p95 | p99 | err

---

signup_storm | 99.8 | 89ms | 188ms | 282ms | 0

login_burst | 321.0 | 18ms | 83ms | 145ms | 0

oauth_client_credentials | 577.6 | 7ms | 78ms | 106ms | 0

token_exchange_chain | 0.0 | 0ms | 0ms | 0ms | 1

cascade_revoke_user_agents | 0.0 | 0ms | 0ms | 0ms | 1

oauth_dpop | 318.9 | 133ms | 209ms | 313ms | 2

rbac_permission_check_hot | 0.0 | 0ms | 0ms | 0ms | 1

vault_read_concurrent | 0.0 | 0ms | 0ms | 0ms | 1

as we can see, performance is competent for now, but for me, this is not enough, and I'm still noticing inconsistencies and low RPS.

Can't provide the full benchmarks for the 0 ms stuff because i'm lazy.

After this benchmark, I implemented a refactor of the signup handler that runs both Create User and Create session so only 1 disk sync.

Earlier on, I used Gemini cli to implement a fast semaphore for argon2id hashing, pretty weak for my standards so I pushed further and guided Gemini on creating a dedicated worker matching VCPUs/cores, enforcing a Fast Timeout with a 429 if queue exceeds a treshold, had a feeling this would improve p99.

After performing a lil refactor here, lil refactor there, this:

go run ./cmd/bench --profile smoke --scenario signup_storm --admin-key-file ../tests/smoke/data/admin.key.firstboot

bench: target=http://localhost:8080 profile=smoke concurrency=10 duration=30s dpop=batched

=== summary (profile: smoke) ===

scenario | rps | p50 | p95 | p99 | err

---

signup_storm | 110.8 | 77ms | 184ms | 271ms | 0

Basically, we achieved a very stable RPS and p99 in the 30 second run, next, was the time of the 2 minutes run.

=== summary (profile: smoke) ===

scenario | rps | p50 | p95 | p99 | err

---

signup_storm | 95.8 | 91ms | 203ms | 314ms | 0

PS C:\Users\raulg\Desktop\projects\shark\bench>

In my opinion, everything is stable enough for me to call it a day.

And after all of this, I still had to finish launch.

RPS optimizations coming on following updates.

# Benchmarks in railway hobby tier 30s 16MB 1 ITER (prod recommendation is 19-32MB, 2 ITER)

- **Signup** → 359 RPS · p99 40ms
- **Login** → 622 RPS · p99 49ms
- **Client Credentials** → 896 RPS · p99 38ms
- **DPOP** → 617 RPS · p99 50ms
- **Cascade Revoke** → 10,600 RPS · p99 11ms
