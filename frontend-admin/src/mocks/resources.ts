import type { ResourceNode } from '../types/resource'

export const resourceTree: ResourceNode[] = [
  {
    id: 'home',
    title: 'Главная',
    type: 'page',
    children: [
      {
        id: 'about',
        title: 'О компании',
        type: 'page',
      },
      {
        id: 'news',
        title: 'Новости',
        type: 'section',
        children: [
          {
            id: 'news-product',
            title: 'Запуск нового продукта',
            type: 'page',
          },
          {
            id: 'news-office',
            title: 'Открытие нового офиса',
            type: 'page',
          },
        ],
      },
      {
        id: 'contacts',
        title: 'Контакты',
        type: 'page',
      },
    ],
  },
  {
    id: 'catalog',
    title: 'Каталог',
    type: 'section',
    children: [
      {
        id: 'services',
        title: 'Услуги',
        type: 'page',
      },
      {
        id: 'cases',
        title: 'Кейсы',
        type: 'page',
      },
      {
        id: 'external-docs',
        title: 'Документация',
        type: 'link',
      },
    ],
  },
]
