"""Finite model checks for proposed progress and pagination laws; no file I/O."""
from itertools import permutations, product

checks = 0
for a, b, c in product(range(8), repeat=3):
    assert max(a, b) == max(b, a)
    assert max(max(a, b), c) == max(a, max(b, c))
    assert max(a, a) == a
    checks += 3
print(f"PASS: {checks} finite checks of max join laws")

posts = [1, 2, 3, 4, 5, 6]
read = 3
expected_unread = {4, 5, 6}
for display in permutations(posts):
    assert {p for p in display if p > read} == expected_unread
print("PASS: immutable-order unread set survives all 720 display permutations")

# A loaded prefix of two is not evidence for a high pinned post at position six.
loaded = {1, 2, 6}
assert max(loaded) == 6 and not set(range(1, 7)).issubset(loaded)
print("COUNTEREXAMPLE: loaded {1,2,6}; max=6 falsely acknowledges unseen {3,4,5}")

# A finite high watermark prevents concurrent appends extending this traversal.
snapshot = 6
cursor = 0
emitted = []
live = posts.copy()
while True:
    page = [p for p in live if cursor < p <= snapshot][:2]
    if not page:
        break
    emitted.extend(page)
    cursor = page[-1]
    live.append(max(live) + 1)
assert emitted == posts
print("PASS: snapshot traversal emits 1..6 once despite concurrent appends")

# Multiple reasons must be intersected with the requested scope before choosing
# a display label. Label precedence alone is not an eligibility predicate.
reasons = {"participating", "watching"}
scope = {"watching"}
assert reasons & scope
assert "participating" not in scope
print("COUNTEREXAMPLE: precedence-before-filter loses a valid watching match")
