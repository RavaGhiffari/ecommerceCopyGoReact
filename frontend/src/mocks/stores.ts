export interface StoreRecord {
  id: string
  slug: string
  name: string
  description: string
  location: string
  ratingAverage: number
  ratingCount: number
  joinedAt: string
}

export const stores: StoreRecord[] = [
  {
    id: 'store-1',
    slug: 'gadget-nusantara',
    name: 'Gadget Nusantara',
    description: 'Aksesoris dan perangkat elektronik harian dengan garansi toko 7 hari.',
    location: 'Jakarta Barat',
    ratingAverage: 4.8,
    ratingCount: 9420,
    joinedAt: '2023-03-14T00:00:00Z',
  },
  {
    id: 'store-2',
    slug: 'batik-kita',
    name: 'Batik Kita',
    description: 'Batik dan busana tradisional modern langsung dari pengrajin Solo.',
    location: 'Surabaya',
    ratingAverage: 4.7,
    ratingCount: 6180,
    joinedAt: '2022-08-02T00:00:00Z',
  },
  {
    id: 'store-3',
    slug: 'rumah-asri',
    name: 'Rumah Asri',
    description: 'Perlengkapan rumah, dapur, dan dekorasi untuk hunian yang nyaman.',
    location: 'Bandung',
    ratingAverage: 4.6,
    ratingCount: 7350,
    joinedAt: '2024-01-21T00:00:00Z',
  },
  {
    id: 'store-4',
    slug: 'sportivo-store',
    name: 'Sportivo Store',
    description: 'Peralatan olahraga dan buku hobi untuk gaya hidup aktif.',
    location: 'Yogyakarta',
    ratingAverage: 4.7,
    ratingCount: 3890,
    joinedAt: '2023-11-05T00:00:00Z',
  },
]
