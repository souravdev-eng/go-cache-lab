# 04: Replicated hot key

**What to build:** A learner can read the designated popular book through a route that stores several logical copies in Redis and selects among them for reads. The route and guide make the selected copy observable, so the learner can inspect distribution across keys.

**Blocked by:** 01: Runnable bookstore lab.

**Status:** ready-for-agent

- [ ] The replica count is configurable and greater than one in the learning setup.
- [ ] A cold read creates or finds the needed logical copies; repeated reads select among more than one copy and return the same book data.
- [ ] Copies use a distinct namespace and a configurable TTL.
- [ ] Unknown books remain 404 and do not create replicas.
- [ ] HTTP-level acceptance checks verify correct responses and distribution across logical copies.
- [ ] The guide explains that multiple keys on one Redis instance do not spread load across Redis nodes.
