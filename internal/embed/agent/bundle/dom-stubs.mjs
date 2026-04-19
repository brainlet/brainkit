// Tiny DOM-surface stubs required by pdfjs-dist for text extraction.
// pdfjs references DOMMatrix / Path2D / ImageData at module load time;
// QuickJS has none of them. Full fidelity isn't needed because we only
// call getDocument(...).getTextContent() which returns raw strings.
if (typeof globalThis.DOMMatrix === 'undefined') {
  class DOMMatrix {
    constructor(init) {
      this.a = 1; this.b = 0; this.c = 0; this.d = 1; this.e = 0; this.f = 0;
      this.m11 = 1; this.m12 = 0; this.m13 = 0; this.m14 = 0;
      this.m21 = 0; this.m22 = 1; this.m23 = 0; this.m24 = 0;
      this.m31 = 0; this.m32 = 0; this.m33 = 1; this.m34 = 0;
      this.m41 = 0; this.m42 = 0; this.m43 = 0; this.m44 = 1;
      this.is2D = true;
      this.isIdentity = true;
      if (Array.isArray(init) && init.length === 6) {
        [this.a, this.b, this.c, this.d, this.e, this.f] = init;
        this.m11 = this.a; this.m12 = this.b;
        this.m21 = this.c; this.m22 = this.d;
        this.m41 = this.e; this.m42 = this.f;
        this.isIdentity = this.a === 1 && this.b === 0 && this.c === 0 && this.d === 1 && this.e === 0 && this.f === 0;
      }
    }
    multiply(o) {
      const a = this.a * o.a + this.c * o.b;
      const b = this.b * o.a + this.d * o.b;
      const c = this.a * o.c + this.c * o.d;
      const d = this.b * o.c + this.d * o.d;
      const e = this.a * o.e + this.c * o.f + this.e;
      const f = this.b * o.e + this.d * o.f + this.f;
      return new DOMMatrix([a, b, c, d, e, f]);
    }
    multiplySelf(o) {
      const r = this.multiply(o);
      this.a = r.a; this.b = r.b; this.c = r.c; this.d = r.d; this.e = r.e; this.f = r.f;
      return this;
    }
    translate(tx, ty) { return this.multiply(new DOMMatrix([1, 0, 0, 1, tx, ty])); }
    translateSelf(tx, ty) { return this.multiplySelf(new DOMMatrix([1, 0, 0, 1, tx, ty])); }
    scale(sx, sy) { return this.multiply(new DOMMatrix([sx, 0, 0, sy ?? sx, 0, 0])); }
    scaleSelf(sx, sy) { return this.multiplySelf(new DOMMatrix([sx, 0, 0, sy ?? sx, 0, 0])); }
    inverse() { return new DOMMatrix(); }
    invertSelf() { return this; }
    transformPoint(p) { return { x: (p?.x ?? 0), y: (p?.y ?? 0), z: 0, w: 1 }; }
    toFloat32Array() { return new Float32Array([this.a, this.b, this.c, this.d, this.e, this.f]); }
    toFloat64Array() { return new Float64Array([this.a, this.b, this.c, this.d, this.e, this.f]); }
    toString() { return `matrix(${this.a}, ${this.b}, ${this.c}, ${this.d}, ${this.e}, ${this.f})`; }
  }
  globalThis.DOMMatrix = DOMMatrix;
}
if (typeof globalThis.Path2D === 'undefined') {
  class Path2D {
    constructor() {}
    addPath() {}
    moveTo() {} lineTo() {} arc() {} arcTo() {} closePath() {} rect() {}
    bezierCurveTo() {} quadraticCurveTo() {} ellipse() {}
  }
  globalThis.Path2D = Path2D;
}
if (typeof globalThis.ImageData === 'undefined') {
  class ImageData {
    constructor(w, h) {
      this.width = w || 1;
      this.height = h || 1;
      this.data = new Uint8ClampedArray(this.width * this.height * 4);
    }
  }
  globalThis.ImageData = ImageData;
}
