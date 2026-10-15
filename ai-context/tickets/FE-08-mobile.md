# FE-08. Мобильная версия (< 768px)

**Оценка:** 3 ч · **Зависит от:** FE-04, FE-05, FE-06, FE-07 · **Ветка:** `feature/FE-08-mobile`
**Дизайн:** артборды `MobileMain`, `MobileMainFull`, `MobileFilters`, `MobileCreate`, `EdgeCases`; `design-spec.md` §4, §5

## Цель
Панелью удобно пользоваться с телефона.

## Объём
- Шапка: `text-xl` + тема (`ghost icon size-11`), шапка и строка фильтров `sticky top-0`.
- Плавающая кнопка «+» (`fixed bottom-6 right-4 size-14 rounded-full shadow-lg`), у списка `pb-28`.
- Фильтры: кнопка «Фильтры (N)» (счётчик нестандартных фильтров) + чипы периодов (ToggleGroup `outline`, `overflow-x-auto`, `h-11 rounded-full`) → **Drawer** с ToggleGroup члена семьи / периода / категории, «Выбрать даты», «Сбросить» в шапке, «Применить» внизу (фильтры применяются только по кнопке).
- Сводка: одна Card — «Итого» `text-4xl`, «N трат · в среднем …», stacked BarChart (`layout="vertical"`, высота 12px, обводка 2px `var(--card)`) + три главные доли текстом + Collapsible-легенда «По категориям: N» / «Скрыть категории».
- Список: Item-карточки (`md:hidden`) — иконка категории в круге 40px, комментарий + «Мадина · 30 сен», сумма `shrink-0`, DropdownMenu «⋯» («Изменить», «Удалить» `destructive`, пункты `h-11`).
- «Показать ещё N» (`outline lg w-full`) вместо пагинации — `useExpensesInfinite`, по 10.
- Форма траты во весь экран: `max-md:h-dvh max-md:max-w-none max-md:rounded-none`, сумма первым полем с `autoFocus`, кнопки закреплены внизу.
- Тосты `top-center` на телефоне.
- Все интерактивные элементы ≥ 44px, ввод 16px (без зума в iOS).

## Критерии приёмки
- [ ] На ширине 390px совпадает с артбордами `MobileMain`, `MobileMainFull`, `MobileFilters`, `MobileCreate`; нет горизонтальной прокрутки страницы.
- [ ] Тест: счётчик «Фильтры (N)», «Показать ещё» догружает следующую порцию.
- [ ] Переход через 768px не теряет состояние фильтров.

## Коммит
`feat(frontend): add mobile layout with filter drawer, cards and load more`
