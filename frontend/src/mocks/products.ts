// Fixture rows shaped like database rows (store_id, category_id, status, ...),
// NOT like API responses. src/services turns them into the API shapes in
// src/types/marketplace.ts. Nothing outside src/services imports this file.

export interface VariantRecord {
  id: string
  sku: string
  name: string
  price: number
  stock: number
}

export interface ProductRecord {
  id: string
  storeId: string
  categoryId: string
  slug: string
  name: string
  description: string
  status: 'active' | 'archived'
  ratingAverage: number
  ratingCount: number
  soldCount: number
  createdAt: string
  variants: VariantRecord[]
}

// A fixed reference date keeps the fixtures deterministic (no Date.now()).
const BASE = Date.parse('2026-11-01T00:00:00Z')
const daysAgo = (n: number) => new Date(BASE - n * 86_400_000).toISOString()

// [slug, name, storeId, categoryId, description, variants[name, price, stock],
//  ratingAverage, ratingCount, soldCount, createdDaysAgo, status?]
type Row = [
  string, string, string, string, string,
  [string, number, number][],
  number, number, number, number,
  ('active' | 'archived')?,
]

const rows: Row[] = [
  // ---- Gadget Nusantara -----------------------------------------------------
  ['earbuds-bluetooth-pro', 'Earbuds Bluetooth Pro ANC', 'store-1', 'cat-elektronik',
    'Earbuds nirkabel dengan peredam bising aktif, baterai hingga 30 jam bersama case, dan mikrofon jernih untuk panggilan.',
    [['Hitam', 349000, 40], ['Putih', 349000, 25]], 4.8, 1250, 3200, 12],
  ['power-bank-20000', 'Power Bank 20.000 mAh Fast Charging', 'store-1', 'cat-elektronik',
    'Kapasitas besar dengan dua port USB dan satu USB-C 22,5W. Cocok untuk perjalanan jauh.',
    [['20.000 mAh', 259000, 120]], 4.7, 860, 5100, 40],
  ['smartwatch-fit-2', 'Smartwatch Fit 2 Layar AMOLED', 'store-1', 'cat-elektronik',
    'Layar AMOLED 1,43 inci, pemantau detak jantung, tahan air 5 ATM, dan 100+ mode olahraga.',
    [['Hitam', 599000, 15], ['Silver', 629000, 0]], 4.5, 430, 980, 8],
  ['keyboard-mekanikal-75', 'Keyboard Mekanikal 75% Hot-swap', 'store-1', 'cat-elektronik',
    'Layout 75% ringkas, switch dapat diganti tanpa solder, tersedia koneksi kabel dan Bluetooth.',
    [['Brown Switch', 499000, 22], ['Blue Switch', 489000, 10]], 4.9, 310, 740, 20],
  ['mouse-wireless-silent', 'Mouse Wireless Silent Click', 'store-1', 'cat-elektronik',
    'Klik senyap, sensor 1600 DPI, dan baterai tahan hingga 12 bulan.',
    [['Hitam', 129000, 200]], 4.6, 2100, 8900, 90],
  ['kabel-usbc-braided', 'Kabel USB-C Braided', 'store-1', 'cat-elektronik',
    'Kabel anyaman nilon anti kusut, mendukung pengisian cepat 60W dan transfer data.',
    [['1 meter', 39000, 300], ['2 meter', 59000, 180]], 4.4, 5200, 21000, 150],

  // ---- Batik Kita -----------------------------------------------------------
  ['kemeja-batik-pria-parang', 'Kemeja Batik Pria Motif Parang', 'store-2', 'cat-fashion-pria',
    'Batik cap katun halus dengan motif parang klasik. Nyaman dipakai untuk acara formal maupun santai.',
    [['M', 189000, 18], ['L', 189000, 24], ['XL', 199000, 9]], 4.8, 640, 2300, 25],
  ['kaos-polos-cotton-24s', 'Kaos Polos Cotton Combed 24s', 'store-2', 'cat-fashion-pria',
    'Bahan cotton combed 24s yang adem dan tidak mudah melar. Tersedia banyak warna dasar.',
    [['M', 59000, 80], ['L', 59000, 95], ['XL', 65000, 40]], 4.6, 3800, 17500, 60],
  ['gamis-batik-modern', 'Gamis Batik Modern Rayon', 'store-2', 'cat-fashion-wanita',
    'Gamis rayon adem dengan kombinasi batik modern, dilengkapi saku dan resleting depan busui friendly.',
    [['S', 259000, 12], ['M', 259000, 20], ['L', 259000, 14]], 4.7, 520, 1500, 18],
  ['tunik-bordir-elegan', 'Tunik Bordir Elegan', 'store-2', 'cat-fashion-wanita',
    'Tunik bordir halus di bagian dada dan lengan, jatuh nyaman dan mudah dipadukan.',
    [['All size', 175000, 33]], 4.5, 260, 640, 33],
  ['sarung-tenun-premium', 'Sarung Tenun Premium', 'store-2', 'cat-fashion-pria',
    'Sarung tenun halus berbahan katun, warna tidak luntur, cocok untuk ibadah dan hadiah.',
    [['Motif Kotak', 145000, 60]], 4.9, 150, 410, 5],
  ['hijab-segi-empat-voal', 'Hijab Segi Empat Voal Premium', 'store-2', 'cat-fashion-wanita',
    'Voal ultrafine yang ringan dan mudah dibentuk, jahit tepi rapi dan tidak menerawang.',
    [['Dusty Pink', 49000, 150], ['Sage', 49000, 120], ['Hitam', 49000, 300]], 4.6, 2900, 12300, 70],
  ['kemeja-batik-edisi-lama', 'Kemeja Batik Edisi Lama (arsip)', 'store-2', 'cat-fashion-pria',
    'Produk yang sudah diarsipkan penjual; tidak boleh muncul di katalog publik.',
    [['M', 99000, 5]], 4.0, 12, 30, 400, 'archived'],

  // ---- Rumah Asri -----------------------------------------------------------
  ['set-panci-anti-lengket', 'Set Panci Anti Lengket 5 Pcs', 'store-3', 'cat-rumah-tangga',
    'Lapisan anti lengket bebas PFOA, cocok untuk kompor gas maupun induksi.',
    [['5 Pcs', 459000, 30]], 4.7, 710, 1900, 30],
  ['rak-sepatu-susun-4', 'Rak Sepatu Susun 4 Tingkat', 'store-3', 'cat-rumah-tangga',
    'Rangka besi kokoh dengan rak tertutup. Muat hingga 12 pasang sepatu.',
    [['Putih', 189000, 45]], 4.4, 980, 3400, 45],
  ['lampu-meja-led-dimmable', 'Lampu Meja LED Dimmable', 'store-3', 'cat-rumah-tangga',
    '3 mode warna cahaya, 5 tingkat kecerahan, dan port USB untuk mengisi daya ponsel.',
    [['Putih', 135000, 70]], 4.6, 420, 1250, 15],
  ['diffuser-aromaterapi', 'Diffuser Aromaterapi Ultrasonik', 'store-3', 'cat-rumah-tangga',
    'Tangki 300 ml dengan lampu malam, mati otomatis saat air habis.',
    [['Kayu', 175000, 0]], 4.5, 330, 870, 22],
  ['sprei-katun-queen', 'Sprei Katun Queen Size', 'store-3', 'cat-rumah-tangga',
    'Katun halus 160x200 cm dengan tinggi kasur hingga 30 cm, termasuk dua sarung bantal.',
    [['Motif Daun', 249000, 25], ['Motif Garis', 249000, 31]], 4.8, 1100, 4200, 55],
  ['botol-minum-stainless-750', 'Botol Minum Stainless 750ml', 'store-3', 'cat-rumah-tangga',
    'Menahan dingin hingga 24 jam dan panas hingga 12 jam. Tutup anti bocor.',
    [['Hijau', 89000, 140], ['Hitam', 89000, 180]], 4.7, 3600, 14000, 100],
  ['puzzle-1000-pemandangan', 'Puzzle 1000 Keping Pemandangan', 'store-3', 'cat-buku-hobi',
    'Puzzle karton tebal dengan hasil cetak tajam, ukuran jadi 70x50 cm.',
    [['1000 keping', 119000, 28]], 4.6, 190, 520, 48],

  // ---- Sportivo Store -------------------------------------------------------
  ['sepatu-lari-ultralight', 'Sepatu Lari Ultralight', 'store-4', 'cat-olahraga',
    'Bobot hanya 230 gram dengan midsole responsif dan upper rajut yang bernapas.',
    [['40', 549000, 6], ['41', 549000, 11], ['42', 549000, 14], ['43', 549000, 5]], 4.8, 740, 2100, 10],
  ['matras-yoga-tpe-8mm', 'Matras Yoga TPE 8mm', 'store-4', 'cat-olahraga',
    'Permukaan anti selip dua sisi, ramah lingkungan, dilengkapi tali pengikat.',
    [['Ungu', 149000, 55], ['Biru', 149000, 48]], 4.6, 1500, 5300, 38],
  ['dumbbell-adjustable-20kg', 'Dumbbell Adjustable 2x10kg', 'store-4', 'cat-olahraga',
    'Beban dapat diatur dari 2,5 sampai 10 kg per sisi, pegangan karet anti selip.',
    [['2 x 10 kg', 699000, 8]], 4.7, 240, 520, 28],
  ['raket-badminton-karbon', 'Raket Badminton Karbon', 'store-4', 'cat-olahraga',
    'Frame karbon ringan 85 gram dengan keseimbangan head-heavy untuk smash tajam.',
    [['Hitam-Merah', 329000, 19]], 4.5, 410, 1100, 42],
  ['buku-latihan-konsisten-30-hari', 'Buku Latihan Konsisten 30 Hari', 'store-4', 'cat-buku-hobi',
    'Panduan latihan harian dengan lembar pelacak kebiasaan dan catatan progres.',
    [['Paperback', 79000, 90]], 4.9, 130, 360, 3],
]

export const products: ProductRecord[] = rows.map(
  ([slug, name, storeId, categoryId, description, variants, ratingAverage, ratingCount, soldCount, created, status], i) => {
    const id = `prod-${String(i + 1).padStart(2, '0')}`
    return {
      id,
      storeId,
      categoryId,
      slug,
      name,
      description,
      status: status ?? 'active',
      ratingAverage,
      ratingCount,
      soldCount,
      createdAt: daysAgo(created),
      variants: variants.map(([variantName, price, stock], v) => ({
        id: `${id}-v${v + 1}`,
        sku: `${slug.slice(0, 6).toUpperCase()}-${String(v + 1).padStart(2, '0')}`,
        name: variantName,
        price,
        stock,
      })),
    }
  },
)
