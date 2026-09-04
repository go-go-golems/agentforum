[Accessibility](https://accessibility.mit.edu)

### 6.033--Computer System Engineering

#### Suggestions for classroom discussion of:

J\[erome\] H. Saltzer, D\[avid\]. P. Reed, and D\[avid\]. D. Clark. End-to-end arguments in system design.*ACM Transactions on Computer Systems 2,* 4 (November, 1984) pages 277-288.

by J. H. Saltzer, March 17, 1996

---

```
This paper, being about a design principle, is best discussed in the
context of other case studies.

1.  Can you find any examples in the systems we have read about where
the end-to-end argument was applied, or ought to be applied?

     - Ethernet:  Is the CRC necessary or a good idea?
       (originally intended to help detect collisions and discard runt
       packets.  Probably not important any more.)

     - How does IP/TCP reflect the end-to-end argument?  (The original
       1974 proposal for internetworking provided an application
       interface that had only a TCP-like function.  Dave Reed
       successfully argued that application access to a datagram-only
       service was essential, so TCP/IP was split into two layers and
       applications were given the option of using the IP layer
       directly.)

    -  Alternate link management in Autonet:  failover to the alternate
       link is done by the hardware interface, at the physical level of
       the link layer) rather than referring things to the network layer
       and letting it decide to use a different physical link.  Is this a
       good idea?  (contradicts E/E argument.  Perhaps the network layer
       has a better failover strategy.  But in this case it seems very
       likely that referring the problem to the network layer would just
       result in the same decision at the higher cost of requiring that
       that layer know too much about how Autonet works.

     - Birrell RPC doesn't provide timeout for server.  Why?  (Only the
       client knows what is a reasonable time to wait or whether timeouts
       are even appropriate.)

     - X did not use RPC.  How does the end-to-end argument apply here?
       (This one is more subtle than it appears.  In contrast with the
       ISO/OSI networking model, the end-to-end argument was used in the
       design of TCP/IP to insist that the
       application have direct access to both layers, rather than
       requiring that the application go through a presentation layer
       such as RPC.  Thus X was able to design its own protocol.

2.   Pronet Token Ring versus IBM Token Ring:  Pronet has only a parity
check and it is recalculated at each repeater, so target recipient may
not even find out that there was an error on the other side of the
ring.  IBM design passes unchanged CRC from station to station; an
error anywhere along the line will be immediately evident to the
recipient of a packet. Can you find an end-to-end argument that
suggests which design is preferable? (A ring uses point-to-point, not
broadcast, links, which rarely fail in subtle ways.  Either they work
very well or they blow out completely, in which case loss of token will
tell you.  Damaged packets are so rare it doesn't help higher layers to
report them. The parity check is designed as a fault isolation tool
which helps point to a link that is starting to fail. The IBM design
tried to help the next-higher layer by providing a port-to-port check.
Higher layers then tried to avoid providing their own checksums and got
into trouble.  The most common trouble was when a link in the return
path between the target and the originator did start to go flaky, the
IBM ring turns into a powerful duplicate generator.)

3.  Another approach:  can you find non-network examples of the
end-to-end argument?

     - frame the argument for micro-kernels as an end-to-end argument.
       (Stuff crammed into the kernel must be used by everyone, whether
       it meets their needs or not; things outside the kernel can be
       easily tailored or replaced.)

     - frame the argument for RISC versus CISC architectures as
       an end-to-end argument.
       (Giving the application a complex instruction that is
       close to, but not exactly, what is needed is costly, both for that
       instruction implementation and for the machine as a whole.  A
       lower level interface allows the program to design its own complex
       instructions that are tailored to its real requirements.)

     - etc.

3.  These last items suggest that it can be interesting to identify
arguments that go in the other direction.  There are good (performance)
reasons for using monolithic kernels;.  There are always pressures to
increase the function of RISC processors. RPC is popular.  Why?

4.  Just how does the packet voice example make use of the end-to-end
argument?  (This example, which appears as a cryptic footnote on page
282,  is worth exploring in depth.  For packet voice, the most
important consideration is regular delivery of successive packets.  If
a packet hasn't arrived in time, or has been damaged, the receiving end
can insert 100 ms of low-level noise into the reconstructed sound
stream with minimum disruption to the ongoing conversation.  Chances are
that there is enough redundancy in the higher-level language that it
won't matter, and if it does matter, the speakers already have a polished
retransmission protocol in which one says "what?".  The bottom line here
is that if lower levels of the network have increased delay variability
at all in order to reduce packet damage or improve delivery assurance,
they are probably working against the best interests of this application.

5.  What causes duplicate messages?  (Someone at a lower level has timed
out and resent a message, even though the original wasn't actually
lost, it was just delayed more than the timeout.  Another cause:  a
token ring can create duplicates.  If a packet is damaged after it
passes the target but before it returns to the originator, the
originator will see the damage and resend that packet.  The token rings
used in the NFSNet backbone gateways were notorious for duplicating
(and losing) packets. On bad days one would send 1000 packets to
California and the recipient would receive 1015 packets, of which only
970 were different.  3% were lost, 5% were duplicated.)

It is very hard to choose correct timeouts. Why is it hard?  (Because
things not under your control affect the optimum timeout value.)

6.  Is it OK for lower levels to do a bad job?  (No.  They should do the
best they can to the extent that they aren't usurping tradeoffs better
made by higher levels.)

[The following suggestions for discussing this paper come from
Deborah Estrin, when she was a recitation instructor in 6.033.]

7.  For the careful file transfer example, identify the threats to
successful completion.  For each threat, identify the approaches taken
to achieve greater reliability (reinforce each step, end-to-end check,
make communication medium more reliable).

8.  Given the end-to-end argument, is there any reason to do error
detection and correction at lower levels?

6.  Illustrate the overhead of a virtual circuit versus a datagram
service by bringing along and waving copies of the specification
documents for TCP and UDP. (This suggestion is more interesting after
lectures on the end-to-end layer has been given.)

9.  There are some potential political issues regarding the end-to-end
argument as applied to networks.  In long-haul situations, the network
is provided by the common carrier, while the nodes belong to users.  In
those countries where the common carrier is restricted from entering
other businesses, what position would you expect the common carrier to
take on end-to-end issues such as reliability, deduplication, and
packet ordering?
```

---

Comments and suggestions: Saltzer@mit.edu