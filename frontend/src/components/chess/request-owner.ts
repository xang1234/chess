export type RequestToken = {
  controller: AbortController
  id: number
}

export class RequestOwner {
  private active: RequestToken | undefined
  private sequence = 0

  start(): RequestToken {
    this.active?.controller.abort()
    const controller = new AbortController()
    this.sequence += 1
    this.active = { controller, id: this.sequence }
    return this.active
  }

  cancel(): void {
    this.active?.controller.abort()
    this.active = undefined
    this.sequence += 1
  }

  isCurrent(token: RequestToken): boolean {
    return !token.controller.signal.aborted && this.active?.controller === token.controller &&
      this.active.id === token.id && this.sequence === token.id
  }
}
