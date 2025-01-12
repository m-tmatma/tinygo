//go:build rp2040

package machine

import (
	"device/rp"
	"runtime/volatile"
	"unsafe"
)

const (
	_NUMBANK0_GPIOS  = 30
	_NUMBANK0_IRQS   = 4
	_NUMIRQ          = 32
	rp2350ExtraReg   = 0
	RESETS_RESET_Msk = 0x01ffffff
	initUnreset      = rp.RESETS_RESET_ADC |
		rp.RESETS_RESET_RTC |
		rp.RESETS_RESET_SPI0 |
		rp.RESETS_RESET_SPI1 |
		rp.RESETS_RESET_UART0 |
		rp.RESETS_RESET_UART1 |
		rp.RESETS_RESET_USBCTRL
	initDontReset = rp.RESETS_RESET_IO_QSPI |
		rp.RESETS_RESET_PADS_QSPI |
		rp.RESETS_RESET_PLL_USB |
		rp.RESETS_RESET_USBCTRL |
		rp.RESETS_RESET_SYSCFG |
		rp.RESETS_RESET_PLL_SYS
	padEnableMask = rp.PADS_BANK0_GPIO0_IE_Msk |
		rp.PADS_BANK0_GPIO0_OD_Msk
)

const (
	PinOutput PinMode = iota
	PinInput
	PinInputPulldown
	PinInputPullup
	PinAnalog
	PinUART
	PinPWM
	PinI2C
	PinSPI
	PinPIO0
	PinPIO1
)

const (
	ClkGPOUT0 clockIndex = iota // GPIO Muxing 0
	ClkGPOUT1                   // GPIO Muxing 1
	ClkGPOUT2                   // GPIO Muxing 2
	ClkGPOUT3                   // GPIO Muxing 3
	ClkRef                      // Watchdog and timers reference clock
	ClkSys                      // Processors, bus fabric, memory, memory mapped registers
	ClkPeri                     // Peripheral clock for UART and SPI
	ClkUSB                      // USB clock
	ClkADC                      // ADC clock
	ClkRTC                      // Real time clock
	NumClocks
)

func CalcClockDiv(srcFreq, freq uint32) uint32 {
	// Div register is 24.8 int.frac divider so multiply by 2^8 (left shift by 8)
	return uint32((uint64(srcFreq) << 8) / uint64(freq))
}

type clocksType struct {
	clk   [NumClocks]clockType
	resus struct {
		ctrl   volatile.Register32
		status volatile.Register32
	}
	fc0      fc
	wakeEN0  volatile.Register32
	wakeEN1  volatile.Register32
	sleepEN0 volatile.Register32
	sleepEN1 volatile.Register32
	enabled0 volatile.Register32
	enabled1 volatile.Register32
	intR     volatile.Register32
	intE     volatile.Register32
	intF     volatile.Register32
	intS     volatile.Register32
}

// GPIO function selectors
const (
	fnJTAG pinFunc = 0
	fnSPI  pinFunc = 1 // Connect one of the internal PL022 SPI peripherals to GPIO
	fnUART pinFunc = 2
	fnI2C  pinFunc = 3
	// Connect a PWM slice to GPIO. There are eight PWM slices,
	// each with two outputchannels (A/B). The B pin can also be used as an input,
	// for frequency and duty cyclemeasurement
	fnPWM pinFunc = 4
	// Software control of GPIO, from the single-cycle IO (SIO) block.
	// The SIO function (F5)must be selected for the processors to drive a GPIO,
	// but the input is always connected,so software can check the state of GPIOs at any time.
	fnSIO pinFunc = 5
	// Connect one of the programmable IO blocks (PIO) to GPIO. PIO can implement a widevariety of interfaces,
	// and has its own internal pin mapping hardware, allowing flexibleplacement of digital interfaces on bank 0 GPIOs.
	// The PIO function (F6, F7) must beselected for PIO to drive a GPIO, but the input is always connected,
	// so the PIOs canalways see the state of all pins.
	fnPIO0, fnPIO1 pinFunc = 6, 7
	// General purpose clock inputs/outputs. Can be routed to a number of internal clock domains onRP2040,
	// e.g. Input: to provide a 1 Hz clock for the RTC, or can be connected to an internalfrequency counter.
	// e.g. Output: optional integer divide
	fnGPCK pinFunc = 8
	// USB power control signals to/from the internal USB controller
	fnUSB  pinFunc = 9
	fnNULL pinFunc = 0x1f

	fnXIP pinFunc = 0
)

// Configure configures the gpio pin as per mode.
func (p Pin) Configure(config PinConfig) {
	if p == NoPin {
		return
	}
	p.init()
	mask := uint32(1) << p
	switch config.Mode {
	case PinOutput:
		p.setFunc(fnSIO)
		rp.SIO.GPIO_OE_SET.Set(mask)
	case PinInput:
		p.setFunc(fnSIO)
		p.pulloff()
	case PinInputPulldown:
		p.setFunc(fnSIO)
		p.pulldown()
	case PinInputPullup:
		p.setFunc(fnSIO)
		p.pullup()
	case PinAnalog:
		p.setFunc(fnNULL)
		p.pulloff()
	case PinUART:
		p.setFunc(fnUART)
	case PinPWM:
		p.setFunc(fnPWM)
	case PinI2C:
		// IO config according to 4.3.1.3 of rp2040 datasheet.
		p.setFunc(fnI2C)
		p.pullup()
		p.setSchmitt(true)
		p.setSlew(false)
	case PinSPI:
		p.setFunc(fnSPI)
	case PinPIO0:
		p.setFunc(fnPIO0)
	case PinPIO1:
		p.setFunc(fnPIO1)
	}
}

var (
	timer = (*timerType)(unsafe.Pointer(rp.TIMER))
)

// Enable or disable a specific interrupt on the executing core.
// num is the interrupt number which must be in [0,31].
func irqSet(num uint32, enabled bool) {
	if num >= _NUMIRQ {
		return
	}
	irqSetMask(1<<num, enabled)
}

func irqSetMask(mask uint32, enabled bool) {
	if enabled {
		// Clear pending before enable
		// (if IRQ is actually asserted, it will immediately re-pend)
		rp.PPB.NVIC_ICPR.Set(mask)
		rp.PPB.NVIC_ISER.Set(mask)
	} else {
		rp.PPB.NVIC_ICER.Set(mask)
	}
}

func (clks *clocksType) initRTC() {
	// ClkRTC = pllUSB (48MHz) / 1024 = 46875Hz
	clkrtc := clks.clock(ClkRTC)
	clkrtc.configure(0, // No GLMUX
		rp.CLOCKS_CLK_RTC_CTRL_AUXSRC_CLKSRC_PLL_USB,
		48*MHz,
		46875)
}

func (clks *clocksType) initTicks() {} // No ticks on RP2040

// startTick starts the watchdog tick.
// cycles needs to be a divider that when applied to the xosc input,
// produces a 1MHz clock. So if the xosc frequency is 12MHz,
// this will need to be 12.
func (wd *watchdogImpl) startTick(cycles uint32) {
	rp.WATCHDOG.TICK.Set(cycles | rp.WATCHDOG_TICK_ENABLE)
}
