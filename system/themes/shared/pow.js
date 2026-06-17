// ── PoW (Proof of Work) — computation engine + UI glue ────────────

(function () {
  // ═══════════════════════════════════════════════════════════════════
  // Computation engine
  // ═══════════════════════════════════════════════════════════════════

  // ── Bit counting (on raw bytes, not hex) ──────────────────────────
  function countLeadingZeroBitsFromBytes(bytes) {
    var count = 0;
    for (var i = 0; i < bytes.length; i++) {
      var b = bytes[i];
      if (b === 0) {
        count += 8;
        continue;
      }
      if (b & 0x80) break;
      if (b & 0x40) {
        count += 1;
        break;
      }
      if (b & 0x20) {
        count += 2;
        break;
      }
      if (b & 0x10) {
        count += 3;
        break;
      }
      if (b & 0x08) {
        count += 4;
        break;
      }
      if (b & 0x04) {
        count += 5;
        break;
      }
      if (b & 0x02) {
        count += 6;
        break;
      }
      if (b & 0x01) {
        count += 7;
        break;
      }
      count += 8;
      break;
    }
    return count;
  }

  // ── Solver ─────────────────────────────────────────────────────────
  // challenge: string, difficulty: number (bits)
  // Returns a Promise that resolves to the winning nonce.
  async function solve(challenge, difficulty) {
    var prefix = challenge + ":";
    var nonce = 0;

    // Fast path: native crypto.subtle (available on all modern browsers)
    var encoder;
    try {
      if (
        typeof crypto !== "undefined" &&
        crypto.subtle &&
        crypto.subtle.digest
      ) {
        encoder = new TextEncoder();
      }
    } catch (e) {}
    if (encoder) {
      while (true) {
        var data = encoder.encode(prefix + nonce);
        var hash = new Uint8Array(await crypto.subtle.digest("SHA-256", data));
        if (countLeadingZeroBitsFromBytes(hash) >= difficulty) return nonce;
        nonce++;
      }
    }

    // ── Compact SHA-256 (FIPS 180-4) —─────────────────────────────────
    // Fallback for older browsers without crypto.subtle.
    function S(x, n) {
      return (x >>> n) | (x << (32 - n));
    }
    function R(x, n) {
      return x >>> n;
    }
    function Ch(x, y, z) {
      return (x & y) ^ (~x & z);
    }
    function Maj(x, y, z) {
      return (x & y) ^ (x & z) ^ (y & z);
    }
    function S0(x) {
      return S(x, 2) ^ S(x, 13) ^ S(x, 22);
    }
    function S1(x) {
      return S(x, 6) ^ S(x, 11) ^ S(x, 25);
    }
    function s0(x) {
      return S(x, 7) ^ S(x, 18) ^ R(x, 3);
    }
    function s1(x) {
      return S(x, 17) ^ S(x, 19) ^ R(x, 10);
    }
    function add(x, y) {
      return ((x + y) | 0) >>> 0;
    }

    var K = [
      0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1,
      0x923f82a4, 0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
      0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786,
      0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
      0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147,
      0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
      0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
      0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
      0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a,
      0x5b9cca4f, 0x682e6ff3, 0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
      0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
    ];

    var H = [
      0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c,
      0x1f83d9ab, 0x5be0cd19,
    ];
    var nextUpdate = 0;

    function sha256(msg) {
      var blen = msg.length * 8;
      var M = [];
      for (var i = 0; i < msg.length; i++)
        M[i >> 2] =
          M[i >> 2] | 0 | ((msg.charCodeAt(i) & 0xff) << (24 - (i % 4) * 8));
      M[blen >> 5] = M[blen >> 5] | 0 | (0x80 << (24 - (blen % 32)));
      M[(((blen + 64) >>> 9) << 4) + 15] = blen;

      var W = new Array(64);
      for (var i = 0; i < M.length; i += 16) {
        var a = H[0],
          b = H[1],
          c = H[2],
          d = H[3],
          e = H[4],
          f = H[5],
          g = H[6],
          h = H[7];
        for (var t = 0; t < 64; t++) {
          if (t < 16) W[t] = M[i + t] | 0;
          else
            W[t] = add(
              add(add(s1(W[t - 2]), W[t - 7]), s0(W[t - 15])),
              W[t - 16],
            );
          var T1 = add(add(add(add(h, S1(e)), Ch(e, f, g)), K[t]), W[t]);
          var T2 = add(S0(a), Maj(a, b, c));
          h = g;
          g = f;
          f = e;
          e = add(d, T1);
          d = c;
          c = b;
          b = a;
          a = add(T1, T2);
        }
        H[0] = add(a, H[0]);
        H[1] = add(b, H[1]);
        H[2] = add(c, H[2]);
        H[3] = add(d, H[3]);
        H[4] = add(e, H[4]);
        H[5] = add(f, H[5]);
        H[6] = add(g, H[6]);
        H[7] = add(h, H[7]);
      }

      var hex = "";
      for (var i = 0; i < 8; i++)
        hex += ("00000000" + (H[i] >>> 0).toString(16)).slice(-8);
      return hex;
    }

    function countLeadingZeroBits(hex) {
      var count = 0;
      for (var i = 0; i < hex.length; i++) {
        var nibble = parseInt(hex[i], 16);
        if (nibble === 0) {
          count += 4;
          continue;
        }
        if (nibble & 8) break;
        if (nibble & 4) {
          count += 1;
          break;
        }
        if (nibble & 2) {
          count += 2;
          break;
        }
        if (nibble & 1) {
          count += 3;
          break;
        }
        count += 4;
        break;
      }
      return count;
    }

    while (true) {
      var hash = sha256(prefix + nonce);
      var zeros = countLeadingZeroBits(hash);
      if (zeros >= difficulty) return nonce;
      nonce++;
      if (nonce % 8192 === 0 && Date.now() > nextUpdate) {
        await new Promise(function (r) {
          setTimeout(r, 0);
        });
        nextUpdate = Date.now() + 80;
      }
    }
  }

  // ═══════════════════════════════════════════════════════════════════
  // UI glue — auto-bootstraps from DOM data attributes
  // ═══════════════════════════════════════════════════════════════════

  var wrapper = document.getElementById("pow-wrapper");
  if (wrapper) {
    var challenge = wrapper.getAttribute("data-challenge");
    var difficulty = parseInt(wrapper.getAttribute("data-difficulty")) || 20;
    var errorEl = document.getElementById("pow-error");
    var isRunning = false;

    window.startPow = function () {
      if (isRunning) return;
      isRunning = true;
      document.getElementById("pow-start").className = "pow-hidden";
      document.getElementById("pow-verifying").className = "";
      // Yield to the browser so it paints the "verifying" UI before starting computation
      setTimeout(function () {
        solve(challenge, difficulty)
          .then(function (nonce) {
            document.getElementById("pow-verifying").className = "pow-hidden";
            document.getElementById("pow-done").className = "";
            document.getElementById("nonce-input").value = nonce;
            setTimeout(function () {
              document.getElementById("pow-form").submit();
            }, 1200);
          })
          .catch(function (err) {
            document.getElementById("pow-verifying").className = "pow-hidden";
            errorEl.textContent = '{{ __ "pow_error" }}';
            errorEl.className = "pow-error";
            console.error("PoW solver error:", err);
          });
      }, 50);
    };
  }
})();
