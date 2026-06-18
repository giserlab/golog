// PoW (Proof of Work) computation engine and UI glue.
(function () {
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

  var H0 = [
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c,
    0x1f83d9ab, 0x5be0cd19,
  ];

  var W = new Array(64);

  function rotr(x, n) {
    return (x >>> n) | (x << (32 - n));
  }

  function add2(a, b) {
    return (a + b) >>> 0;
  }

  function add4(a, b, c, d) {
    return (a + b + c + d) >>> 0;
  }

  function add5(a, b, c, d, e) {
    return (a + b + c + d + e) >>> 0;
  }

  function decimalLength(n) {
    if (n < 10) return 1;
    if (n < 100) return 2;
    if (n < 1000) return 3;
    if (n < 10000) return 4;
    if (n < 100000) return 5;
    if (n < 1000000) return 6;
    if (n < 10000000) return 7;
    if (n < 100000000) return 8;
    if (n < 1000000000) return 9;
    return String(n).length;
  }

  function byteAt(prefix, prefixLen, nonce, nonceLen, index) {
    if (index < prefixLen) return prefix.charCodeAt(index) & 0xff;

    var pos = index - prefixLen;
    var div = 1;
    for (var i = pos + 1; i < nonceLen; i++) div *= 10;
    return 48 + Math.floor(nonce / div) % 10;
  }

  // Returns the first 32 bits of SHA256(prefix + nonce). Difficulty is capped
  // at 30 in settings, so the first word is enough for the leading-zero test.
  function sha256FirstWord(prefix, nonce) {
    var prefixLen = prefix.length;
    var nonceLen = decimalLength(nonce);
    var msgLen = prefixLen + nonceLen;
    var bitLen = msgLen * 8;
    var blockCount = (((msgLen + 9 + 63) / 64) | 0);

    var h0 = H0[0];
    var h1 = H0[1];
    var h2 = H0[2];
    var h3 = H0[3];
    var h4 = H0[4];
    var h5 = H0[5];
    var h6 = H0[6];
    var h7 = H0[7];

    for (var block = 0; block < blockCount; block++) {
      for (var i = 0; i < 16; i++) {
        var word = 0;
        for (var j = 0; j < 4; j++) {
          var msgIndex = block * 64 + i * 4 + j;
          var value = 0;
          if (msgIndex < msgLen) {
            value = byteAt(prefix, prefixLen, nonce, nonceLen, msgIndex);
          } else if (msgIndex === msgLen) {
            value = 0x80;
          } else if (block === blockCount - 1 && msgIndex >= block * 64 + 56) {
            var lenIndex = msgIndex - (block * 64 + 56);
            value = lenIndex < 4 ? 0 : (bitLen >>> ((7 - lenIndex) * 8)) & 0xff;
          }
          word = (word << 8) | value;
        }
        W[i] = word >>> 0;
      }

      for (var t = 16; t < 64; t++) {
        var s0 = rotr(W[t - 15], 7) ^ rotr(W[t - 15], 18) ^ (W[t - 15] >>> 3);
        var s1 = rotr(W[t - 2], 17) ^ rotr(W[t - 2], 19) ^ (W[t - 2] >>> 10);
        W[t] = add4(W[t - 16], s0, W[t - 7], s1);
      }

      var a = h0;
      var b = h1;
      var c = h2;
      var d = h3;
      var e = h4;
      var f = h5;
      var g = h6;
      var h = h7;

      for (var round = 0; round < 64; round++) {
        var S1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
        var ch = (e & f) ^ (~e & g);
        var temp1 = add5(h, S1, ch, K[round], W[round]);
        var S0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
        var maj = (a & b) ^ (a & c) ^ (b & c);
        var temp2 = add2(S0, maj);

        h = g;
        g = f;
        f = e;
        e = add2(d, temp1);
        d = c;
        c = b;
        b = a;
        a = add2(temp1, temp2);
      }

      h0 = add2(h0, a);
      h1 = add2(h1, b);
      h2 = add2(h2, c);
      h3 = add2(h3, d);
      h4 = add2(h4, e);
      h5 = add2(h5, f);
      h6 = add2(h6, g);
      h7 = add2(h7, h);
    }

    return h0 >>> 0;
  }

  function hasLeadingZeroBits(firstWord, difficulty) {
    if (difficulty <= 0) return true;
    if (difficulty >= 32) return firstWord === 0;
    return (firstWord >>> (32 - difficulty)) === 0;
  }

  async function solve(challenge, difficulty) {
    var workerCount = Math.min(Math.max(navigator.hardwareConcurrency || 2, 2), 4);
    if (workerCount > 1 && typeof Worker !== "undefined" && typeof Blob !== "undefined" && typeof URL !== "undefined") {
      try {
        return await solveWithWorkers(challenge, difficulty, workerCount);
      } catch (e) {
        console.warn("PoW worker solver failed, falling back to single thread:", e);
      }
    }

    return solveSingleThread(challenge, difficulty, 0, 1);
  }

  async function solveSingleThread(challenge, difficulty, startNonce, step) {
    var prefix = challenge + ":";
    var nonce = startNonce;
    var batchSize = 16384;

    while (true) {
      for (var i = 0; i < batchSize; i++, nonce += step) {
        if (hasLeadingZeroBits(sha256FirstWord(prefix, nonce), difficulty)) {
          return nonce;
        }
      }

      await new Promise(function (resolve) {
        setTimeout(resolve, 0);
      });
    }
  }

  function solveWithWorkers(challenge, difficulty, workerCount) {
    return new Promise(function (resolve, reject) {
      var script = [
        "var K=" + JSON.stringify(K) + ";",
        "var H0=" + JSON.stringify(H0) + ";",
        "var W=new Array(64);",
        rotr.toString(),
        add2.toString(),
        add4.toString(),
        add5.toString(),
        decimalLength.toString(),
        byteAt.toString(),
        sha256FirstWord.toString(),
        hasLeadingZeroBits.toString(),
        "self.onmessage=function(e){",
        "var prefix=e.data.challenge+':';",
        "var difficulty=e.data.difficulty;",
        "var nonce=e.data.startNonce;",
        "var step=e.data.step;",
        "while(true){",
        "for(var i=0;i<16384;i++,nonce+=step){",
        "if(hasLeadingZeroBits(sha256FirstWord(prefix,nonce),difficulty)){self.postMessage({nonce:nonce});return;}",
        "}",
        "}",
        "};",
      ].join("\n");

      var url = URL.createObjectURL(new Blob([script], { type: "application/javascript" }));
      var workers = [];
      var settled = false;

      function cleanup() {
        for (var i = 0; i < workers.length; i++) workers[i].terminate();
        URL.revokeObjectURL(url);
      }

      for (var i = 0; i < workerCount; i++) {
        var worker = new Worker(url);
        workers.push(worker);
        worker.onmessage = function (event) {
          if (settled) return;
          settled = true;
          cleanup();
          resolve(event.data.nonce);
        };
        worker.onerror = function (event) {
          if (settled) return;
          settled = true;
          cleanup();
          reject(event.error || new Error(event.message || "worker error"));
        };
        worker.postMessage({
          challenge: challenge,
          difficulty: difficulty,
          startNonce: i,
          step: workerCount,
        });
      }
    });
  }

  var wrapper = document.getElementById("pow-wrapper");
  if (wrapper) {
    var challenge = wrapper.getAttribute("data-challenge");
    var difficulty = parseInt(wrapper.getAttribute("data-difficulty"), 10) || 20;
    var errorEl = document.getElementById("pow-error");
    var isRunning = false;

    window.startPow = function () {
      if (isRunning) return;
      isRunning = true;
      document.getElementById("pow-start").className = "pow-hidden";
      document.getElementById("pow-verifying").className = "";

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
