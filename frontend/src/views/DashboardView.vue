<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import AppIcon from '../components/AppIcon.vue'
import { api } from '../api.js'

const router = useRouter()
const loading = ref(true)
const sidebarOpen = ref(false)
const activePage = ref('home')
const toast = ref('')
const teachers = ref([])
const students = ref([])
const bookings = ref([])
const slots = ref([])
const slotsLoading = ref(false)
const bookingSlotEditorOpen = ref(false)
const bookingSlotEditorSaving = ref(false)
const bookingSubmitting = ref(false)
const selectedBookingSlot = ref(null)
const availability = ref([])
const links = ref([])
const profileLinks = ref([])
const users = ref([])
const realUser = ref(null)
const calendarTeacherId = ref('')
const linkStudentId = ref('')
const calendarAnchor = ref('')
const calendarScroll = ref(null)
const profileSaving = ref(false)
const lessonLinks = ref([])
const lessonLinksBookingId = ref('')
const lessonLinksOpen = ref(false)
const lessonLinksLoading = ref(false)

const HOUR_HEIGHT = 56
const CALENDAR_START_HOUR = 0
const CALENDAR_END_HOUR = 24
const calendarGridHeight = (CALENDAR_END_HOUR - CALENDAR_START_HOUR) * HOUR_HEIGHT

const localDate = (offsetDays = 0) => {
  const date = new Date(Date.now() + offsetDays * 86400000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 10)
}

const bookingForm = ref({ teacherId: '', duration: 60 })
const bookingSlotForm = ref({ date: '', time: '' })
const availabilityForm = ref({ teacherId: '', date: localDate(3), from: '15:00', to: '21:00', durations: [30, 60] })
const userForm = ref({ fullName: '', login: '', password: '', role: 'student', timezone: Intl.DateTimeFormat().resolvedOptions().timeZone })
const profileForm = ref({ fullName: '', subjectLabel: '' })

const currentUser = computed(() => realUser.value)
const role = computed(() => currentUser.value?.role || 'student')
const firstName = computed(() => currentUser.value?.fullName?.split(' ')[0] || 'пользователь')

const navByRole = computed(() => {
  const common = [{ id: 'home', label: 'Главная', icon: 'home' }, { id: 'schedule', label: 'Расписание', icon: 'calendar' }, { id: 'profile', label: 'Профиль', icon: 'user' }]
  if (role.value === 'student') return [...common, { id: 'book', label: 'Записаться', icon: 'plus' }]
  if (role.value === 'teacher') return [...common, { id: 'availability', label: 'Свободное время', icon: 'clock' }, { id: 'links', label: 'Конференции', icon: 'link' }]
  return [...common, { id: 'users', label: 'Пользователи', icon: 'users' }, { id: 'availability', label: 'Свободное время', icon: 'clock' }, { id: 'links', label: 'Конференции', icon: 'link' }]
})

const visibleBookings = computed(() => {
  if (role.value === 'student') return bookings.value.filter(item => item.studentId === currentUser.value?.id)
  if (role.value === 'teacher') return bookings.value.filter(item => item.teacherId === currentUser.value?.id)
  return bookings.value
})
const pendingBookings = computed(() => visibleBookings.value.filter(item => item.status === 'pending'))
const upcomingBookings = computed(() => visibleBookings.value
  .filter(item => ['pending', 'confirmed'].includes(item.status) && new Date(item.startsAt) >= new Date())
  .sort((left, right) => new Date(left.startsAt) - new Date(right.startsAt))
  .slice(0, 4))
const bookingSlotRanges = computed(() => {
  const duration = bookingForm.value.duration
  const eligible = slots.value
    .filter(item => item.allowedDurations?.includes(duration))
    .sort((left, right) => new Date(left.startsAt) - new Date(right.startsAt))
  const ranges = []
  for (const slot of eligible) {
    const previous = ranges[ranges.length - 1]
    const previousSlot = previous?.slotOptions[previous.slotOptions.length - 1]
    const consecutive = previous
      && previous.availabilityId === slot.availabilityId
      && formatISODate(previous.startsAt) === formatISODate(slot.startsAt)
      && new Date(slot.startsAt) - new Date(previousSlot.startsAt) === 10 * 60000
    if (consecutive) {
      previous.slotOptions.push(slot)
      previous.lastStartsAt = slot.startsAt
      previous.endsAt = new Date(new Date(slot.startsAt).getTime() + duration * 60000).toISOString()
      continue
    }
    ranges.push({
      id: `${slot.availabilityId}-${slot.startsAt}`,
      availabilityId: slot.availabilityId,
      startsAt: slot.startsAt,
      lastStartsAt: slot.startsAt,
      endsAt: new Date(new Date(slot.startsAt).getTime() + duration * 60000).toISOString(),
      slotOptions: [slot],
      kind: 'slot',
    })
  }
  return ranges
})
const selectedBookingCalendarItem = computed(() => {
  const slot = selectedBookingSlot.value
  if (!slot) return null
  return {
    ...slot,
    endsAt: new Date(new Date(slot.startsAt).getTime() + bookingForm.value.duration * 60000).toISOString(),
    kind: 'selected-slot',
  }
})
const calendarHours = Array.from({ length: CALENDAR_END_HOUR - CALENDAR_START_HOUR }, (_, index) => index + CALENDAR_START_HOUR)
const calendarBookings = computed(() => {
  if (role.value === 'moderator' && calendarTeacherId.value) {
    return visibleBookings.value.filter(item => item.teacherId === calendarTeacherId.value)
  }
  return visibleBookings.value
})
const weekDays = computed(() => {
  const start = startOfWeekKey(calendarAnchor.value || localDate())
  const today = currentUser.value ? formatISODate(new Date()) : localDate()
  return Array.from({ length: 7 }, (_, index) => {
    const key = addDaysToKey(start, index)
    const date = new Date(`${key}T12:00:00Z`)
    return {
      key,
      weekday: new Intl.DateTimeFormat('ru-RU', { weekday: 'short', timeZone: 'UTC' }).format(date).replace('.', ''),
      day: new Intl.DateTimeFormat('ru-RU', { day: '2-digit', timeZone: 'UTC' }).format(date),
      month: new Intl.DateTimeFormat('ru-RU', { month: 'short', timeZone: 'UTC' }).format(date).replace('.', ''),
      today: key === today,
    }
  })
})
const weekLabel = computed(() => {
  const first = weekDays.value[0]
  const last = weekDays.value[6]
  if (!first || !last) return ''
  return `${first.day} ${first.month} — ${last.day} ${last.month}`
})
const timezoneLabel = computed(() => currentUser.value?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone)
const miniWeekDays = computed(() => {
  const start = startOfWeekKey(formatISODate(new Date()))
  return ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'].map((label, index) => ({ label, key: addDaysToKey(start, index) }))
})
const currentWeekBookings = computed(() => {
  const keys = new Set(miniWeekDays.value.map(day => day.key))
  return visibleBookings.value.filter(item => ['pending', 'confirmed'].includes(item.status) && keys.has(formatISODate(item.startsAt)))
})

function notify(message) {
  toast.value = message
  window.setTimeout(() => { toast.value = '' }, 2800)
}

function teacherName(id) { return teachers.value.find(item => item.id === id)?.fullName || 'Преподаватель' }
function teacherSubject(id) { return teachers.value.find(item => item.id === id)?.subjectLabel || 'Предмет не указан' }
function studentName(id) { return students.value.find(item => item.id === id)?.fullName || 'Ученик' }
function formatDate(value, options = {}) {
  return new Intl.DateTimeFormat('ru-RU', { timeZone: currentUser.value?.timezone || 'Europe/Moscow', ...options }).format(new Date(value))
}
function formatISODate(value) {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: currentUser.value?.timezone || 'Europe/Moscow',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date(value))
  const byType = Object.fromEntries(parts.map(part => [part.type, part.value]))
  return `${byType.year}-${byType.month}-${byType.day}`
}
function addDaysToKey(key, days) {
  const date = new Date(`${key}T12:00:00Z`)
  date.setUTCDate(date.getUTCDate() + days)
  return date.toISOString().slice(0, 10)
}
function startOfWeekKey(key) {
  const date = new Date(`${key}T12:00:00Z`)
  const offset = (date.getUTCDay() + 6) % 7
  return addDaysToKey(key, -offset)
}
function timeParts(value) {
  const parts = new Intl.DateTimeFormat('en-GB', {
    timeZone: currentUser.value?.timezone || 'Europe/Moscow',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).formatToParts(new Date(value))
  const byType = Object.fromEntries(parts.map(part => [part.type, part.value]))
  return { hour: Number(byType.hour), minute: Number(byType.minute) }
}
function calendarItemDuration(item) {
  if (item.kind === 'booking') return item.durationMinutes
  return Math.max(10, Math.round((new Date(item.endsAt) - new Date(item.startsAt)) / 60000))
}
function calendarItemStyle(item) {
  const start = timeParts(item.startsAt)
  const startMinutes = start.hour * 60 + start.minute
  const visibleStart = Math.max(startMinutes, CALENDAR_START_HOUR * 60)
  const visibleEnd = Math.min(startMinutes + calendarItemDuration(item), CALENDAR_END_HOUR * 60)
  if (visibleEnd <= visibleStart) return { display: 'none' }
  return {
    top: `${((visibleStart - CALENDAR_START_HOUR * 60) / 60) * HOUR_HEIGHT}px`,
    height: `${Math.max(24, ((visibleEnd - visibleStart) / 60) * HOUR_HEIGHT - 2)}px`,
  }
}
function calendarItemsForDay(dayKey) {
  const bookingItems = calendarBookings.value
    .filter(item => ['pending', 'confirmed'].includes(item.status) && formatISODate(item.startsAt) === dayKey)
    .map(item => ({ ...item, kind: 'booking' }))
  const availabilityItems = role.value === 'student'
    ? []
    : availability.value
      .filter(item => item.teacherId === calendarTeacherId.value && formatISODate(item.startsAt) === dayKey)
      .map(item => ({ ...item, kind: 'availability' }))
  return [...availabilityItems, ...bookingItems]
}
function bookingSlotRangesForDay(dayKey) {
  return bookingSlotRanges.value.filter(item => formatISODate(item.startsAt) === dayKey)
}
function calendarItemTitle(item) {
  if (item.kind === 'availability') return 'Свободное время'
  return role.value === 'student' ? teacherName(item.teacherId) : studentName(item.studentId)
}
function calendarItemTime(item) {
  return `${formatDate(item.startsAt, { hour: '2-digit', minute: '2-digit' })}–${formatDate(item.endsAt, { hour: '2-digit', minute: '2-digit' })}`
}
function timeInputValue(value) {
  const time = timeParts(value)
  return `${String(time.hour).padStart(2, '0')}:${String(time.minute).padStart(2, '0')}`
}
function bookingSlotFormValue(slot) {
  if (!slot) return { date: '', time: '' }
  return { date: formatISODate(slot.startsAt), time: timeInputValue(slot.startsAt) }
}
function openBookingSlotEditor(range, event) {
  const column = event.currentTarget.parentElement
  if (!column || !range.slotOptions.length) return
  const bounds = column.getBoundingClientRect()
  const clickedMinutes = CALENDAR_START_HOUR * 60 + ((event.clientY - bounds.top) / HOUR_HEIGHT) * 60
  const selected = range.slotOptions.reduce((closest, candidate) => {
    const candidateTime = timeParts(candidate.startsAt)
    const closestTime = timeParts(closest.startsAt)
    const candidateDistance = Math.abs(candidateTime.hour * 60 + candidateTime.minute - clickedMinutes)
    const closestDistance = Math.abs(closestTime.hour * 60 + closestTime.minute - clickedMinutes)
    return candidateDistance < closestDistance ? candidate : closest
  })
  bookingSlotForm.value = bookingSlotFormValue(selected)
  bookingSlotEditorOpen.value = true
}
function closeBookingSlotEditor() {
  if (bookingSlotEditorSaving.value) return
  bookingSlotForm.value = bookingSlotFormValue(selectedBookingSlot.value)
  bookingSlotEditorOpen.value = false
}
function moveWeek(days) {
  calendarAnchor.value = addDaysToKey(calendarAnchor.value, days)
}
function goToCurrentWeek() {
  const today = formatISODate(new Date())
  if (calendarAnchor.value === today) {
    if (activePage.value === 'book') loadSlots()
    else loadCalendarAvailability()
  } else {
    calendarAnchor.value = today
  }
  scrollCalendarToWorkingHours()
}
function scrollCalendarToWorkingHours() {
  nextTick(() => {
    if (calendarScroll.value) calendarScroll.value.scrollTop = 7 * HOUR_HEIGHT
  })
}
function zonedLocalToUTC(date, time) {
  const timezone = currentUser.value?.timezone || 'Europe/Moscow'
  const naiveUTC = new Date(`${date}T${time}:00Z`)
  const zoneName = new Intl.DateTimeFormat('en', {
    timeZone: timezone,
    timeZoneName: 'longOffset',
  }).formatToParts(naiveUTC).find(part => part.type === 'timeZoneName')?.value || 'GMT'
  const match = zoneName.match(/GMT([+-])(\d{2}):(\d{2})/)
  const offset = match ? (match[1] === '+' ? 1 : -1) * (Number(match[2]) * 60 + Number(match[3])) : 0
  return new Date(naiveUTC.getTime() - offset * 60000).toISOString()
}
function statusLabel(status) {
  return ({ pending: 'Ожидает подтверждения', confirmed: 'Подтверждено', rejected: 'Отклонено', cancelled_by_student: 'Отменено', cancelled_by_staff: 'Отменено' })[status] || status
}

async function toggleLessonLinks(booking) {
  if (lessonLinksOpen.value && lessonLinksBookingId.value === booking.id) {
    lessonLinksOpen.value = false
    return
  }

  lessonLinksBookingId.value = booking.id
  lessonLinks.value = []
  lessonLinksOpen.value = true
  lessonLinksLoading.value = true
  try {
    const response = await api.bookingConferenceLinks(booking.id)
    if (lessonLinksBookingId.value === booking.id) lessonLinks.value = response.items || []
  } catch {
    if (lessonLinksBookingId.value === booking.id) lessonLinks.value = []
    notify('Не удалось загрузить ссылки')
  } finally {
    if (lessonLinksBookingId.value === booking.id) lessonLinksLoading.value = false
  }
}

async function load() {
  loading.value = true
  try {
    realUser.value = await api.me()
    profileForm.value.fullName = realUser.value.fullName
    profileForm.value.subjectLabel = realUser.value.subjectLabel || ''
    const [teacherResponse, bookingResponse] = await Promise.all([api.teachers(), api.bookings()])
    teachers.value = teacherResponse.items || []
    bookings.value = bookingResponse.items || []
    bookingForm.value.teacherId = teachers.value[0]?.id || ''
    calendarAnchor.value = formatISODate(new Date())
    calendarTeacherId.value = realUser.value.role === 'teacher' ? realUser.value.id : teachers.value[0]?.id || ''
    availabilityForm.value.teacherId = calendarTeacherId.value
    if (realUser.value.role === 'student') {
      await loadProfileLinks()
    } else {
      students.value = (await api.students()).items || []
      linkStudentId.value = students.value[0]?.id || ''
      await loadStudentLinks()
    }
    if (realUser.value.role === 'moderator') users.value = (await api.users()).items || []
  } catch (err) {
    if (err.status === 401) router.replace('/login')
    else notify('Не удалось загрузить данные')
  } finally {
    loading.value = false
  }
}

async function loadSlots() {
  const teacherId = bookingForm.value.teacherId
  if (!teacherId) {
    slots.value = []
    slotsLoading.value = false
    return
  }
  const weekStart = startOfWeekKey(calendarAnchor.value || localDate())
  const weekEnd = addDaysToKey(weekStart, 7)
  slots.value = []
  slotsLoading.value = true
  try {
    const response = await api.slots(
      teacherId,
      zonedLocalToUTC(weekStart, '00:00'),
      zonedLocalToUTC(weekEnd, '00:00'),
    )
    if (bookingForm.value.teacherId === teacherId && startOfWeekKey(calendarAnchor.value || localDate()) === weekStart) {
      slots.value = response.items || []
    }
  } catch {
    if (bookingForm.value.teacherId === teacherId && startOfWeekKey(calendarAnchor.value || localDate()) === weekStart) {
      notify('Не удалось загрузить свободное время')
    }
  } finally {
    if (bookingForm.value.teacherId === teacherId && startOfWeekKey(calendarAnchor.value || localDate()) === weekStart) {
      slotsLoading.value = false
    }
  }
}

async function loadCalendarAvailability() {
  if (role.value === 'student' || !calendarTeacherId.value) {
    availability.value = []
    return
  }
  const teacherId = calendarTeacherId.value
  const weekStart = startOfWeekKey(calendarAnchor.value || localDate())
  const weekEnd = addDaysToKey(weekStart, 7)
  try {
    const response = await api.availability(
      teacherId,
      zonedLocalToUTC(weekStart, '00:00'),
      zonedLocalToUTC(weekEnd, '00:00'),
    )
    if (calendarTeacherId.value === teacherId && startOfWeekKey(calendarAnchor.value || localDate()) === weekStart) {
      availability.value = response.items || []
    }
  } catch {
    if (calendarTeacherId.value === teacherId && startOfWeekKey(calendarAnchor.value || localDate()) === weekStart) {
      notify('Не удалось загрузить свободные интервалы')
    }
  }
}

async function loadStudentLinks() {
  links.value = []
  const studentId = linkStudentId.value
  if (!studentId) return
  try {
    const response = await api.studentConferenceLinks(studentId)
    if (linkStudentId.value === studentId) links.value = response.items || []
  }
  catch { notify('Не удалось загрузить ссылки ученика') }
}

async function loadProfileLinks() {
  profileLinks.value = []
  const studentId = currentUser.value?.id
  if (role.value !== 'student' || !studentId) return
  try {
    const response = await api.studentConferenceLinks(studentId)
    if (currentUser.value?.id === studentId) profileLinks.value = response.items || []
  }
  catch { notify('Не удалось загрузить ссылки для подключения') }
}

async function findAvailableBookingSlot(date, time) {
  if (!bookingForm.value.teacherId || !/^\d{4}-\d{2}-\d{2}$/.test(date) || !/^\d{2}:\d{2}$/.test(time)) return null
  const hour = Number(time.slice(0, 2))
  const minute = Number(time.slice(3, 5))
  if (hour > 23 || minute > 59 || minute % 10 !== 0) return null
  const response = await api.slots(
    bookingForm.value.teacherId,
    zonedLocalToUTC(date, '00:00'),
    zonedLocalToUTC(addDaysToKey(date, 1), '00:00'),
  )
  const target = new Date(zonedLocalToUTC(date, time)).getTime()
  return (response.items || []).find(item => (
    new Date(item.startsAt).getTime() === target
    && item.allowedDurations?.includes(bookingForm.value.duration)
  )) || null
}

async function confirmBookingSlot() {
  const { date, time } = bookingSlotForm.value
  if (!date || !time) return notify('Укажите дату и время начала занятия')
  if (Number(time.slice(3, 5)) % 10 !== 0) return notify('Время начала должно быть кратно 10 минутам')
  bookingSlotEditorSaving.value = true
  try {
    const available = await findAvailableBookingSlot(date, time)
    if (!available) return notify('На выбранную дату и время свободного слота нет')
    selectedBookingSlot.value = available
    bookingSlotEditorOpen.value = false
    if (startOfWeekKey(calendarAnchor.value) !== startOfWeekKey(date)) calendarAnchor.value = date
    notify('Время занятия выбрано')
  } catch {
    notify('Не удалось проверить свободное время')
  } finally {
    bookingSlotEditorSaving.value = false
  }
}

async function submitBooking() {
  if (!selectedBookingSlot.value) return notify('Выберите время занятия на календаре')
  bookingSubmitting.value = true
  try {
    const selected = await findAvailableBookingSlot(bookingSlotForm.value.date, bookingSlotForm.value.time)
    if (!selected) {
      selectedBookingSlot.value = null
      bookingSlotEditorOpen.value = true
      return notify('Выбранное время уже занято. Укажите другое время')
    }
    const created = await api.createBooking({ availabilityId: selected.availabilityId, teacherId: selected.teacherId, startsAt: selected.startsAt, durationMinutes: bookingForm.value.duration })
    bookings.value.unshift(created)
    selectedBookingSlot.value = null
    setPage('schedule')
    notify('Заявка отправлена преподавателю')
  } catch (err) {
    if (err.code === 'conflict' || err.code === 'booking_closed') {
      selectedBookingSlot.value = null
      bookingSlotEditorOpen.value = true
      notify('Выбранное время уже недоступно. Укажите другое время')
    } else {
      notify('Не удалось создать запись')
    }
  } finally {
    bookingSubmitting.value = false
  }
}

async function bookingAction(item, action) {
  try { Object.assign(item, await api.bookingAction(item.id, action)); notify('Статус занятия обновлён') }
  catch { notify('Не удалось изменить занятие') }
}

async function addAvailability() {
  if (!availabilityForm.value.durations.length) return notify('Выберите хотя бы одну длительность')
  const teacherId = role.value === 'teacher' ? currentUser.value.id : availabilityForm.value.teacherId
  if (!teacherId) return notify('Выберите преподавателя')
  try {
    const result = await api.createAvailability({
      teacherId,
      startsAt: zonedLocalToUTC(availabilityForm.value.date, availabilityForm.value.from),
      endsAt: zonedLocalToUTC(availabilityForm.value.date, availabilityForm.value.to),
      allowedDurations: availabilityForm.value.durations,
    })
    const created = result.items || []
    if (calendarTeacherId.value === teacherId) availability.value.push(...created)
    else calendarTeacherId.value = teacherId
    calendarAnchor.value = availabilityForm.value.date
    setPage('schedule')
    if (!created.length) notify('В выбранном диапазоне нет нового свободного времени')
    else if (result.skippedRanges?.length) notify(`Добавлено интервалов: ${created.length}. Пересечения пропущены`)
    else notify(created.length === 1 ? 'Свободное время добавлено' : `Добавлено интервалов: ${created.length}`)
  } catch (err) {
    notify(err.code === 'conflict' ? 'Расписание изменилось. Попробуйте ещё раз' : 'Не удалось добавить интервал')
  }
}

async function saveLinks() {
  const studentId = linkStudentId.value
  if (!studentId) return notify('Выберите ученика')
  for (const item of links.value) {
    item.url = item.url.trim()
    item.active = item.active && Boolean(item.url)
  }
  try {
    await Promise.all(links.value.map(item => api.updateConferenceLink(studentId, item.id, { url: item.url, active: item.active })))
    notify(`Ссылки для ${studentName(studentId)} сохранены`)
  } catch (err) {
    notify(err.code === 'conflict' ? 'Эта ссылка уже закреплена за другим учеником' : 'Не удалось сохранить ссылки')
  }
}

async function createUser() {
  if (!userForm.value.fullName || !userForm.value.login || !userForm.value.password) return notify('Заполните все поля')
  try {
    const created = await api.createUser(userForm.value)
    users.value.unshift(created)
    if (created.role === 'student') {
      students.value.push(created)
      if (!linkStudentId.value) linkStudentId.value = created.id
    }
    notify('Пользователь создан')
  }
  catch { notify('Не удалось создать пользователя') }
}

async function saveProfile() {
  const fullName = profileForm.value.fullName.trim()
  const subjectLabel = profileForm.value.subjectLabel.trim()
  if (!fullName || fullName.length > 200) return notify('Введите имя длиной от 1 до 200 символов')
  if (role.value === 'teacher' && subjectLabel.length > 100) return notify('Подпись предмета не должна превышать 100 символов')
  if (fullName === currentUser.value.fullName && (role.value !== 'teacher' || subjectLabel === currentUser.value.subjectLabel)) return notify('Данные не изменились')
  profileSaving.value = true
  try {
    const payload = { fullName }
    if (role.value === 'teacher') payload.subjectLabel = subjectLabel
    const updated = await api.updateProfile(payload)
    realUser.value = updated
    profileForm.value.fullName = updated.fullName
    profileForm.value.subjectLabel = updated.subjectLabel || ''
    for (const collection of [teachers.value, students.value, users.value]) {
      const item = collection.find(candidate => candidate.id === updated.id)
      if (item) Object.assign(item, updated)
    }
    notify('Профиль сохранён')
  } catch (err) {
    notify(err.code === 'invalid_input' ? 'Проверьте имя и подпись предмета' : 'Не удалось сохранить профиль')
  } finally {
    profileSaving.value = false
  }
}

async function logout() {
  await api.logout().catch(() => {})
  router.push('/')
}

function setPage(page) {
  activePage.value = page
  sidebarOpen.value = false
  if (page === 'schedule') scrollCalendarToWorkingHours()
  if (page === 'book') {
    const today = formatISODate(new Date())
    if (calendarAnchor.value === today) loadSlots()
    else calendarAnchor.value = today
    scrollCalendarToWorkingHours()
  }
  if (page === 'profile' && role.value === 'student') loadProfileLinks()
}

watch(() => bookingForm.value.teacherId, () => {
  selectedBookingSlot.value = null
  bookingSlotEditorOpen.value = false
  if (!loading.value && activePage.value === 'book') loadSlots()
})
watch(() => bookingForm.value.duration, () => {
  selectedBookingSlot.value = null
  bookingSlotEditorOpen.value = false
})
watch(calendarTeacherId, () => {
  if (role.value === 'moderator') availabilityForm.value.teacherId = calendarTeacherId.value
  loadCalendarAvailability()
})
watch(calendarAnchor, () => {
  if (loading.value) return
  if (activePage.value === 'book') loadSlots()
  else loadCalendarAvailability()
})
watch(linkStudentId, () => {
  if (!loading.value) loadStudentLinks()
})
onMounted(load)
</script>

<template>
  <div class="dashboard-shell">
    <aside class="dashboard-sidebar" :class="{ 'dashboard-sidebar--open': sidebarOpen }">
      <div class="sidebar-top">
        <RouterLink class="brand brand--sidebar" to="/"><span class="brand-mark">A</span><span><strong>Александра</strong><small>онлайн-школа</small></span></RouterLink>
        <button class="icon-button sidebar-close" @click="sidebarOpen = false"><AppIcon name="close" /></button>
      </div>
      <nav class="dashboard-nav">
        <button v-for="item in navByRole" :key="item.id" :class="{ active: activePage === item.id }" @click="setPage(item.id)">
          <AppIcon :name="item.icon" :size="20" /><span>{{ item.label }}</span>
        </button>
      </nav>
      <div class="sidebar-quote"><AppIcon name="spark" /><p>Спокойно.<br />Понятно.<br />По шагам.</p></div>
      <button class="logout-button" @click="logout"><AppIcon name="logout" :size="19" /> Выйти</button>
    </aside>

    <div v-if="sidebarOpen" class="sidebar-backdrop" @click="sidebarOpen = false"></div>

    <main class="dashboard-main">
      <header class="dashboard-header">
        <button class="icon-button dashboard-menu" @click="sidebarOpen = true"><AppIcon name="menu" /></button>
        <div><p>{{ role === 'student' ? 'Личный кабинет' : role === 'teacher' ? 'Кабинет преподавателя' : 'Панель модератора' }}</p><h1>Здравствуйте, {{ firstName }}!</h1></div>
        <div class="header-user"><div class="user-avatar">{{ firstName[0] }}</div><span>{{ currentUser?.fullName }}<small>{{ role === 'student' ? 'Ученик' : role === 'teacher' ? 'Преподаватель' : 'Модератор' }}</small></span></div>
      </header>

      <div v-if="loading" class="dashboard-loading"><span></span><p>Собираем расписание…</p></div>

      <div v-else class="dashboard-content">
        <template v-if="activePage === 'home'">
          <section class="welcome-grid">
            <article class="next-lesson-card">
              <div class="card-label"><span></span> Ближайшее занятие</div>
              <template v-if="upcomingBookings.length">
                <p class="lesson-date">{{ formatDate(upcomingBookings[0].startsAt, { weekday: 'long', day: 'numeric', month: 'long' }) }}</p>
                <h2>{{ formatDate(upcomingBookings[0].startsAt, { hour: '2-digit', minute: '2-digit' }) }} <small>· {{ upcomingBookings[0].durationMinutes }} минут</small></h2>
                <p>{{ role === 'student' ? teacherName(upcomingBookings[0].teacherId) : studentName(upcomingBookings[0].studentId) }} · {{ teacherSubject(upcomingBookings[0].teacherId) }}</p>
                <div class="lesson-actions"><div class="lesson-links-control"><button class="button button--white" type="button" :aria-expanded="lessonLinksOpen && lessonLinksBookingId === upcomingBookings[0].id" aria-controls="lesson-links-menu" @click="toggleLessonLinks(upcomingBookings[0])"><AppIcon name="video" :size="18" /> Открыть ссылки</button><div v-if="lessonLinksOpen && lessonLinksBookingId === upcomingBookings[0].id" id="lesson-links-menu" class="lesson-links-menu"><p v-if="lessonLinksLoading" class="lesson-links-empty">Загружаем ссылки…</p><p v-else-if="!lessonLinks.length" class="lesson-links-empty">список пуст</p><template v-else><p v-for="item in lessonLinks" :key="item.id"><span>{{ item.title }}:</span> <a :href="item.url" target="_blank" rel="noopener noreferrer">{{ item.url }}</a></p></template></div></div><span class="status-pill" :class="`status-${upcomingBookings[0].status}`">{{ statusLabel(upcomingBookings[0].status) }}</span></div>
              </template>
              <template v-else><h2>Пока занятий нет</h2><p>Свободное место для новых целей.</p></template>
              <span class="next-formula">a² + b² = c²</span>
            </article>
            <article class="week-card soft-panel">
              <div class="panel-head"><div><span class="card-label">Эта неделя</span><h3>{{ currentWeekBookings.length }} занятия</h3></div><div class="round-icon"><AppIcon name="calendar" /></div></div>
              <div class="mini-days"><span v-for="day in miniWeekDays" :key="day.key" :class="{ filled: currentWeekBookings.some(item => formatISODate(item.startsAt) === day.key) }">{{ day.label }}</span></div>
              <button class="text-button" @click="setPage('schedule')">Открыть расписание <AppIcon name="arrow" :size="16" /></button>
            </article>
          </section>

          <section v-if="role !== 'student' && pendingBookings.length" class="content-section">
            <div class="section-row"><div><p class="section-kicker">Новые заявки</p><h2>Нужно подтвердить</h2></div><span class="count-badge">{{ pendingBookings.length }}</span></div>
            <div class="request-list">
              <article v-for="item in pendingBookings" :key="item.id" class="request-card soft-panel">
                <div class="user-avatar user-avatar--soft">{{ studentName(item.studentId)[0] }}</div>
                <div class="request-main"><strong>{{ studentName(item.studentId) }}</strong><span>{{ formatDate(item.startsAt, { weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) }} · {{ item.durationMinutes }} мин</span></div>
                <div class="request-actions"><button class="circle-action circle-action--ok" title="Подтвердить" @click="bookingAction(item, 'confirm')"><AppIcon name="check" /></button><button class="circle-action" title="Отклонить" @click="bookingAction(item, 'reject')"><AppIcon name="close" /></button></div>
              </article>
            </div>
          </section>

          <section class="content-section quick-section">
            <div class="section-row"><div><p class="section-kicker">Быстрый доступ</p><h2>Что хотите сделать?</h2></div></div>
            <div class="quick-grid">
              <button v-if="role === 'student'" @click="setPage('book')"><span class="round-icon"><AppIcon name="plus" /></span><strong>Записаться</strong><small>Выбрать преподавателя и время</small></button>
              <button v-else @click="setPage('availability')"><span class="round-icon"><AppIcon name="clock" /></span><strong>Добавить время</strong><small>Открыть свободный интервал</small></button>
              <button @click="setPage('schedule')"><span class="round-icon"><AppIcon name="calendar" /></span><strong>Расписание</strong><small>Все ближайшие занятия</small></button>
              <button v-if="role !== 'student'" @click="setPage('links')"><span class="round-icon"><AppIcon name="link" /></span><strong>Конференции</strong><small>Проверить ссылки на урок</small></button>
            </div>
          </section>
        </template>

        <template v-else-if="activePage === 'schedule'">
          <section class="page-title calendar-page-title">
            <div><p class="section-kicker">Ваше время</p><h2>Расписание занятий</h2></div>
            <button v-if="role === 'student'" class="button button--ink" @click="setPage('book')"><AppIcon name="plus" :size="18" /> Записаться</button>
            <button v-else class="button button--ink" @click="setPage('availability')"><AppIcon name="plus" :size="18" /> Добавить время</button>
          </section>

          <section class="calendar-panel soft-panel">
            <div class="calendar-toolbar">
              <div class="calendar-navigation">
                <button class="calendar-nav-arrow" type="button" aria-label="Предыдущая неделя" @click="moveWeek(-7)">‹</button>
                <button class="calendar-today" type="button" @click="goToCurrentWeek">Сегодня</button>
                <button class="calendar-nav-arrow" type="button" aria-label="Следующая неделя" @click="moveWeek(7)">›</button>
                <strong>{{ weekLabel }}</strong>
              </div>
              <div class="calendar-toolbar-right">
                <label v-if="role === 'moderator'" class="calendar-teacher-filter">
                  <span>Преподаватель</span>
                  <select v-model="calendarTeacherId">
                    <option v-for="teacher in teachers" :key="teacher.id" :value="teacher.id">{{ teacher.fullName }}</option>
                  </select>
                </label>
                <div class="calendar-legend">
                  <span><i class="legend-dot legend-dot--lesson"></i> Занятие</span>
                  <span v-if="role !== 'student'"><i class="legend-dot legend-dot--free"></i> Свободно</span>
                </div>
              </div>
            </div>

            <div ref="calendarScroll" class="calendar-scroll">
              <div class="calendar-canvas">
                <div class="calendar-head">
                  <div class="calendar-timezone">{{ timezoneLabel }}</div>
                  <div v-for="day in weekDays" :key="day.key" class="calendar-day-head" :class="{ today: day.today }">
                    <span>{{ day.weekday }}</span><strong>{{ day.day }}</strong><small>{{ day.month }}</small>
                  </div>
                </div>
                <div class="calendar-body" :style="{ height: `${calendarGridHeight}px` }">
                  <div class="calendar-time-axis">
                    <span v-for="hour in calendarHours" :key="hour" :style="{ top: `${(hour - CALENDAR_START_HOUR) * HOUR_HEIGHT}px` }">{{ String(hour).padStart(2, '0') }}:00</span>
                  </div>
                  <div v-for="day in weekDays" :key="day.key" class="calendar-day-column" :class="{ today: day.today }">
                    <article
                      v-for="item in calendarItemsForDay(day.key)"
                      :key="`${item.kind}-${item.id}`"
                      class="calendar-event"
                      :class="[
                        `calendar-event--${item.kind}`,
                        item.kind === 'booking' ? `calendar-event--${item.status}` : '',
                        { 'calendar-event--compact': calendarItemDuration(item) < 45 },
                      ]"
                      :style="calendarItemStyle(item)"
                      :title="`${calendarItemTitle(item)} · ${calendarItemTime(item)}`"
                    >
                      <strong>{{ calendarItemTitle(item) }}</strong>
                      <span>{{ calendarItemTime(item) }}</span>
                      <small v-if="item.kind === 'availability'">Можно записаться на {{ item.allowedDurations.join(' / ') }} мин</small>
                      <small v-else>{{ statusLabel(item.status) }} · онлайн</small>
                      <button
                        v-if="item.kind === 'booking' && ['pending','confirmed'].includes(item.status)"
                        type="button"
                        title="Отменить занятие"
                        aria-label="Отменить занятие"
                        @click.stop="bookingAction(item, 'cancel')"
                      >×</button>
                    </article>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </template>

        <template v-else-if="activePage === 'book'">
          <section class="page-title"><div><p class="section-kicker">Новая запись</p><h2>Выберите время на календаре</h2></div></section>
          <form class="booking-layout" @submit.prevent="submitBooking">
            <div class="booking-main">
              <section class="booking-controls soft-panel">
                <div><div class="form-step"><span>1</span><div><strong>Преподаватель</strong><p>К кому хотите записаться?</p></div></div><div class="teacher-select-grid"><button v-for="teacher in teachers" :key="teacher.id" type="button" :class="{ active: bookingForm.teacherId === teacher.id }" @click="bookingForm.teacherId = teacher.id"><span class="user-avatar">{{ teacher.fullName[0] }}</span><span><strong>{{ teacher.fullName }}</strong><small>{{ teacher.subjectLabel || 'Предмет не указан' }}</small></span><AppIcon v-if="bookingForm.teacherId === teacher.id" name="check" /></button></div></div>
                <div><div class="form-step"><span>2</span><div><strong>Продолжительность</strong><p>Сколько времени понадобится?</p></div></div><div class="choice-row"><button v-for="duration in [15,30,45,60]" :key="duration" type="button" :class="{ active: bookingForm.duration === duration }" @click="bookingForm.duration = duration">{{ duration }} мин</button></div></div>
              </section>

              <section class="calendar-panel booking-calendar soft-panel">
                <div class="calendar-toolbar">
                  <div class="calendar-navigation">
                    <button class="calendar-nav-arrow" type="button" aria-label="Предыдущая неделя" @click="moveWeek(-7)">‹</button>
                    <button class="calendar-today" type="button" @click="goToCurrentWeek">Сегодня</button>
                    <button class="calendar-nav-arrow" type="button" aria-label="Следующая неделя" @click="moveWeek(7)">›</button>
                    <strong>{{ weekLabel }}</strong>
                  </div>
                  <div class="calendar-legend">
                    <span><i class="legend-dot legend-dot--free"></i> Доступно</span>
                    <span><i class="legend-dot legend-dot--selected"></i> Выбрано</span>
                  </div>
                </div>

                <div ref="calendarScroll" class="calendar-scroll">
                  <div class="calendar-canvas">
                    <div class="calendar-head">
                      <div class="calendar-timezone">{{ timezoneLabel }}</div>
                      <div v-for="day in weekDays" :key="day.key" class="calendar-day-head" :class="{ today: day.today }">
                        <span>{{ day.weekday }}</span><strong>{{ day.day }}</strong><small>{{ day.month }}</small>
                      </div>
                    </div>
                    <div class="calendar-body" :style="{ height: `${calendarGridHeight}px` }">
                      <p v-if="!slotsLoading && !bookingSlotRanges.length" class="booking-calendar-empty">На этой неделе нет свободного времени выбранной длительности.</p>
                      <p v-if="slotsLoading" class="booking-calendar-empty">Загружаем свободное время…</p>
                      <div class="calendar-time-axis">
                        <span v-for="hour in calendarHours" :key="hour" :style="{ top: `${(hour - CALENDAR_START_HOUR) * HOUR_HEIGHT}px` }">{{ String(hour).padStart(2, '0') }}:00</span>
                      </div>
                      <div v-for="day in weekDays" :key="day.key" class="calendar-day-column" :class="{ today: day.today }">
                        <button
                          v-for="range in bookingSlotRangesForDay(day.key)"
                          :key="range.id"
                          type="button"
                          class="calendar-event calendar-event--slot"
                          :style="calendarItemStyle(range)"
                          :title="`Доступно ${calendarItemTime(range)}. Нажмите, чтобы выбрать время`"
                          :aria-label="`Выбрать время в интервале ${calendarItemTime(range)}`"
                          @click="openBookingSlotEditor(range, $event)"
                        >
                          <strong>Свободное время</strong>
                          <span>{{ calendarItemTime(range) }}</span>
                          <small>Нажмите, чтобы уточнить время</small>
                        </button>
                        <article
                          v-if="selectedBookingCalendarItem && formatISODate(selectedBookingCalendarItem.startsAt) === day.key"
                          class="calendar-event calendar-event--selected-slot"
                          :class="{ 'calendar-event--compact': bookingForm.duration < 45 }"
                          :style="calendarItemStyle(selectedBookingCalendarItem)"
                        >
                          <strong>Выбрано</strong>
                          <span>{{ calendarItemTime(selectedBookingCalendarItem) }}</span>
                        </article>
                      </div>
                    </div>
                  </div>
                </div>
              </section>
            </div>
            <aside class="booking-summary">
              <p class="hand-note">Почти готово ♡</p><h3>Ваша запись</h3>
              <dl><div><dt>Преподаватель</dt><dd>{{ teacherName(bookingForm.teacherId) }}</dd></div><div><dt>Дата</dt><dd>{{ selectedBookingSlot ? formatDate(selectedBookingSlot.startsAt, { weekday: 'short', day: 'numeric', month: 'short' }) : '—' }}</dd></div><div><dt>Время</dt><dd>{{ selectedBookingCalendarItem ? calendarItemTime(selectedBookingCalendarItem) : '—' }}</dd></div><div><dt>Длительность</dt><dd>{{ bookingForm.duration }} минут</dd></div></dl>
              <button class="button button--ink button--full" type="submit" :disabled="bookingSubmitting">{{ bookingSubmitting ? 'Проверяем время…' : 'Отправить заявку' }} <AppIcon name="arrow" :size="18" /></button>
              <small>{{ selectedBookingSlot ? 'Преподаватель подтвердит запись вручную.' : 'Нажмите на свободный интервал и уточните время в форме.' }}</small>
            </aside>
          </form>
        </template>

        <template v-else-if="activePage === 'profile'">
          <section class="page-title"><div><p class="section-kicker">Личные данные</p><h2>Профиль</h2></div></section>
          <form class="settings-card profile-card soft-panel" @submit.prevent="saveProfile">
            <div class="settings-copy"><div class="round-icon"><AppIcon name="user" :size="28" /></div><h3>Как к вам обращаться</h3><p>Это имя отображается в личном кабинете, расписании и списках участников занятий.</p></div>
            <div class="settings-fields">
              <label class="field-label">Имя и фамилия<input v-model="profileForm.fullName" type="text" required maxlength="200" autocomplete="name" placeholder="Введите имя и фамилию" /></label>
              <label v-if="role === 'teacher'" class="field-label">Подпись предмета<input v-model="profileForm.subjectLabel" type="text" maxlength="100" placeholder="Например, Математика · физика" /><small class="field-hint">Эту подпись увидят ученики при выборе преподавателя.</small></label>
              <button class="button button--ink" type="submit" :disabled="profileSaving">{{ profileSaving ? 'Сохраняем…' : 'Сохранить профиль' }}</button>
            </div>
          </form>
          <section v-if="role === 'student'" class="profile-links-card soft-panel">
            <div class="settings-copy"><div class="round-icon"><AppIcon name="link" :size="28" /></div><h3>Ссылки для подключения</h3><p>Используйте эти ссылки, чтобы подключиться к онлайн-занятию.</p></div>
            <div v-if="profileLinks.length" class="profile-links-list">
              <div v-for="item in profileLinks" :key="item.id" class="profile-link-row">
                <strong>{{ item.title }}:</strong>
                <a :href="item.url" target="_blank" rel="noopener noreferrer">{{ item.url }}</a>
              </div>
            </div>
            <p v-else class="profile-links-empty">Преподаватель добавит ссылки для подключения в ближайшее время.</p>
          </section>
        </template>

        <template v-else-if="activePage === 'availability'">
          <section class="page-title"><div><p class="section-kicker">Расписание</p><h2>Добавить свободное время</h2></div></section>
          <form class="settings-card soft-panel" @submit.prevent="addAvailability">
            <div class="settings-copy"><div class="round-icon"><AppIcon name="clock" :size="28" /></div><h3>Один интервал — много вариантов</h3><p>Например, откройте время с 15:00 до 21:00, а ученик сам выберет удобное начало и длительность.</p></div>
            <div class="settings-fields"><label v-if="role === 'moderator'" class="field-label">Преподаватель<select v-model="availabilityForm.teacherId" required><option v-for="teacher in teachers" :key="teacher.id" :value="teacher.id">{{ teacher.fullName }}</option></select></label><label class="field-label">Дата<input v-model="availabilityForm.date" type="date" /></label><div class="two-fields"><label class="field-label">С<input v-model="availabilityForm.from" type="time" step="600" /></label><label class="field-label">До<input v-model="availabilityForm.to" type="time" step="600" /></label></div><label class="field-label">Доступная длительность</label><div class="choice-row"><label v-for="duration in [15,30,45,60]" :key="duration" class="check-choice"><input v-model="availabilityForm.durations" type="checkbox" :value="duration" /><span>{{ duration }} мин</span></label></div><button class="button button--ink" type="submit"><AppIcon name="plus" :size="18" /> Добавить интервал</button></div>
          </form>
        </template>

        <template v-else-if="activePage === 'links'">
          <section class="page-title"><div><p class="section-kicker">Онлайн-занятия</p><h2>Ссылки учеников</h2></div><div class="link-page-actions"><label class="link-student-filter"><span>Ученик</span><select v-model="linkStudentId"><option v-for="student in students" :key="student.id" :value="student.id">{{ student.fullName }}</option></select></label><button class="button button--ink" :disabled="!linkStudentId" @click="saveLinks">Сохранить изменения</button></div></section>
          <p v-if="linkStudentId" class="links-description">Ссылки закреплены за аккаунтом {{ studentName(linkStudentId) }} и будут доступны на подтверждённых занятиях этого ученика.</p>
          <p v-if="!students.length" class="empty-slots">Сначала создайте аккаунт ученика.</p>
          <div v-else class="link-list">
            <article v-for="item in links" :key="item.id" class="link-row soft-panel">
              <div class="provider-mark">{{ item.title.slice(0, 1) }}</div><div class="link-fields"><label>{{ item.title }}<input v-model="item.url" type="url" placeholder="https://…" /></label></div><label class="toggle"><input v-model="item.active" type="checkbox" :disabled="!item.url" /><span></span><small>{{ item.active ? 'Активна' : 'Скрыта' }}</small></label>
            </article>
          </div>
          <button v-if="linkStudentId" class="add-dashed" @click="notify('Добавление своего сервиса откроется в форме')"><AppIcon name="plus" /> Добавить свой сервис</button>
        </template>

        <template v-else-if="activePage === 'users'">
          <section class="page-title"><div><p class="section-kicker">Управление школой</p><h2>Пользователи</h2></div></section>
          <div class="users-layout">
            <div class="user-table soft-panel"><div class="user-table-head"><span>Пользователь</span><span>Роль</span><span>Статус</span></div><div v-for="item in users" :key="item.id" class="user-table-row"><div><span class="user-avatar user-avatar--soft">{{ item.fullName[0] }}</span><span><strong>{{ item.fullName }}</strong><small>{{ item.login || 'аккаунт' }}</small></span></div><span class="role-chip">{{ item.role === 'student' ? 'Ученик' : item.role === 'teacher' ? 'Учитель' : 'Модератор' }}</span><span class="active-dot"><i></i>{{ item.active === false ? 'Заблокирован' : 'Активен' }}</span></div></div>
            <form class="create-user-card" @submit.prevent="createUser"><p class="hand-note">Новый участник</p><h3>Создать аккаунт</h3><label class="field-label">Имя<input v-model="userForm.fullName" placeholder="Имя и фамилия" /></label><label class="field-label">Логин<input v-model="userForm.login" placeholder="login" /></label><label class="field-label">Временный пароль<input v-model="userForm.password" type="password" placeholder="Не менее 10 символов" /></label><label class="field-label">Роль<select v-model="userForm.role"><option value="student">Ученик</option><option value="teacher">Учитель</option><option value="moderator">Модератор</option></select></label><button class="button button--ink button--full" type="submit">Создать <AppIcon name="arrow" :size="17" /></button></form>
          </div>
        </template>
      </div>
    </main>
    <div v-if="bookingSlotEditorOpen" class="booking-slot-modal" role="presentation" @mousedown.self="closeBookingSlotEditor">
      <form class="booking-slot-dialog" role="dialog" aria-modal="true" aria-labelledby="booking-slot-title" @submit.prevent="confirmBookingSlot">
        <div class="booking-slot-dialog-head">
          <div><p class="section-kicker">Свободный слот</p><h2 id="booking-slot-title">Уточните время занятия</h2></div>
          <button class="icon-button" type="button" aria-label="Закрыть форму" :disabled="bookingSlotEditorSaving" @click="closeBookingSlotEditor"><AppIcon name="close" :size="18" /></button>
        </div>
        <p class="booking-slot-dialog-copy">Можно скорректировать начало занятия. Дата и время будут проверены по актуальному расписанию преподавателя.</p>
        <div class="booking-slot-dialog-fields">
          <label class="field-label">Дата<input v-model="bookingSlotForm.date" type="date" :min="formatISODate(new Date())" required /></label>
          <label class="field-label">Время начала<input v-model="bookingSlotForm.time" type="time" step="600" required /></label>
        </div>
        <dl class="booking-slot-dialog-summary"><div><dt>Преподаватель</dt><dd>{{ teacherName(bookingForm.teacherId) }}</dd></div><div><dt>Длительность</dt><dd>{{ bookingForm.duration }} минут</dd></div><div><dt>Часовой пояс</dt><dd>{{ timezoneLabel }}</dd></div></dl>
        <p class="booking-slot-dialog-note">Время начала выбирается с шагом 10 минут. Занятие должно полностью помещаться в свободный интервал.</p>
        <div class="booking-slot-dialog-actions"><button class="button button--ghost" type="button" :disabled="bookingSlotEditorSaving" @click="closeBookingSlotEditor">Отмена</button><button class="button button--ink" type="submit" :disabled="bookingSlotEditorSaving">{{ bookingSlotEditorSaving ? 'Проверяем…' : 'Выбрать время' }}</button></div>
      </form>
    </div>
    <Transition name="toast"><div v-if="toast" class="toast"><AppIcon name="check" :size="18" />{{ toast }}</div></Transition>
  </div>
</template>
